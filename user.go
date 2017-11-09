package main

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/JormungandrK/microservice-registration/app"
	"github.com/JormungandrK/microservice-registration/config"
	"github.com/JormungandrK/microservice-tools/rabbitmq"
	"github.com/afex/hystrix-go/hystrix"
	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	Config          *config.Config
	ChannelRabbitMQ rabbitmq.Channel
}

// Email holds info for the email template
type Email struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Token string `json:"token,omitempty"`
}

// UserProfile represents User Profle
type UserProfile struct {
	Fullname string
	Email    string
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, config *config.Config, channelRabbitMQ rabbitmq.Channel) *UserController {
	return &UserController{
		Controller:      service.NewController("UserController"),
		Config:          config,
		ChannelRabbitMQ: channelRabbitMQ,
	}
}

// Register runs the register action. It creates a user and user profile.
// Also, it sends a massage to the queue in ordet microservice-mail to send
// varification mail to the user.
func (c *UserController) Register(ctx *app.RegisterUserContext) error {
	user := &app.Users{}
	client := &http.Client{}

	token := generateToken(42)
	ctx.Payload.Token = &token

	// Create new user from payload
	jsonUser, err := json.Marshal(ctx.Payload)
	if err != nil {
		return ctx.InternalServerError(goa.ErrInternal(err))
	}

	output := make(chan *http.Response, 1)
	errorsChan := hystrix.Go("user-microservice.create_user", func() error {
		resp, err := makeRequest(client, http.MethodPost, jsonUser, c.Config.Services["user-microservice"], c.Config)
		if err != nil {
			return err
		}
		output <- resp
		return nil
	}, nil)

	var createUserResp *http.Response
	select {
	case out := <-output:
		createUserResp = out
	case respErr := <-errorsChan:
		return ctx.InternalServerError(goa.ErrInternal(respErr))
	}

	body, err := ioutil.ReadAll(createUserResp.Body)
	if err != nil {
		return ctx.InternalServerError(goa.ErrInternal(err))
	}

	if createUserResp.StatusCode != 200 && createUserResp.StatusCode != 201 {
		goaErr := &goa.ErrorResponse{}

		err = json.Unmarshal(body, goaErr)
		if err != nil {
			ctx.InternalServerError(goa.ErrInternal(err))
		}

		switch createUserResp.StatusCode {
		case 400:
			return ctx.BadRequest(goaErr)
		case 500:
			return ctx.InternalServerError(goaErr)
		}
	}

	if err = json.Unmarshal(body, &user); err != nil {
		return ctx.InternalServerError(goa.ErrInternal(err))
	}

	// Update user profile. Create it if does not exist
	user.Fullname = ctx.Payload.Fullname
	userProfile := UserProfile{user.Fullname, user.Email}
	jsonUseProfile, err := json.Marshal(userProfile)
	if err != nil {
		return ctx.InternalServerError(goa.ErrInternal(err))
	}

	upOutput := make(chan *http.Response, 1)
	upErrorChan := hystrix.Go("user-microservice.update_user_profile", func() error {
		resp, errUserProfile := makeRequest(client, http.MethodPut, jsonUseProfile, fmt.Sprintf("%s/%s", c.Config.Services["microservice-user-profile"], user.ID), c.Config)
		if errUserProfile != nil {
			return errUserProfile
		}
		upOutput <- resp
		return nil
	}, nil)

	var createUpResp *http.Response
	select {
	case out := <-upOutput:
		createUpResp = out
	case respErr := <-upErrorChan:
		return ctx.InternalServerError(goa.ErrInternal(respErr))
	}

	body, err = ioutil.ReadAll(createUpResp.Body)
	if err != nil {
		return ctx.InternalServerError(goa.ErrInternal(err))
	}

	if createUpResp.StatusCode != 200 && createUpResp.StatusCode != 204 {
		goaErr := &goa.ErrorResponse{}

		err = json.Unmarshal(body, goaErr)
		if err != nil {
			ctx.InternalServerError(goa.ErrInternal(err))
		}

		switch createUpResp.StatusCode {
		case 400:
			return ctx.BadRequest(goaErr)
		case 500:
			return ctx.InternalServerError(goaErr)
		}
	}

	if ctx.Payload.ExternalID == nil {
		emailInfo := Email{user.ID, user.Fullname, user.Email, token}
		body, err := json.Marshal(emailInfo)
		if err != nil {
			return ctx.InternalServerError(goa.ErrInternal(err))
		}

		if err := c.ChannelRabbitMQ.Send("verification-email", body); err != nil {
			return ctx.InternalServerError(goa.ErrInternal(err))
		}
	}

	return ctx.Created(user)
}

// makeRequest makes http request
func makeRequest(client *http.Client, method string, payload []byte, url string, cfg *config.Config) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	token, err := selfSignJWT(cfg)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, err
}

// selfSignJWT generates a JWT token which is self-signed with the system private key.
// This token is used for accesing the user and user-profile microservices.
func selfSignJWT(cfg *config.Config) (string, error) {
	key, err := ioutil.ReadFile(cfg.SystemKey)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	claims := jwtgo.MapClaims{
		"iss":      "microservice-registration",
		"exp":      time.Now().Add(time.Duration(30) * time.Second).Unix(),
		"jti":      uuid.NewV4().String(),
		"nbf":      0,
		"sub":      "microservice-registration",
		"scope":    "api:read",
		"userId":   "system",
		"username": "system",
		"roles":    "system",
	}

	tokenRS := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims)
	tokenStr, err := tokenRS.SignedString(privateKey)

	return tokenStr, err
}

func generateToken(n int) string {
	rv := make([]byte, n)
	if _, err := rand.Reader.Read(rv); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(rv)
}
