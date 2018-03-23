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

	"github.com/Microkubes/microservice-registration/app"
	"github.com/Microkubes/microservice-registration/config"
	"github.com/Microkubes/microservice-tools/rabbitmq"
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
	Client          *http.Client
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
func NewUserController(service *goa.Service, config *config.Config, channelRabbitMQ rabbitmq.Channel, client *http.Client) *UserController {
	return &UserController{
		Controller:      service.NewController("UserController"),
		Config:          config,
		ChannelRabbitMQ: channelRabbitMQ,
		Client:          client,
	}
}

// Register runs the register action. It creates a user and user profile.
// Also, it sends a massage to the queue in ordet microservice-mail to send
// varification mail to the user.
func (c *UserController) Register(ctx *app.RegisterUserContext) error {
	user := &app.Users{}

	token := generateToken(42)
	ctx.Payload.Token = &token

	// Create new user from payload
	jsonUser, err := json.Marshal(ctx.Payload)
	if err != nil {
		return ctx.InternalServerError(goa.ErrInternal(err))
	}

	output := make(chan *http.Response, 1)
	errorsChan := hystrix.Go("user-microservice.create_user", func() error {
		resp, e := makeRequest(c.Client, http.MethodPost, jsonUser, c.Config.Services["user-microservice"], c.Config)
		if e != nil {
			return e
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
		resp, errUserProfile := makeRequest(c.Client, http.MethodPut, jsonUseProfile, fmt.Sprintf("%s/%s", c.Config.Services["microservice-user-profile"], user.ID), c.Config)
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

// ResendVerification resets the activation token and resends activation emal to user.
func (c *UserController) ResendVerification(ctx *app.ResendVerificationUserContext) error {
	// 1. Reset user token
	userID, token, err := c.resetVerificationToken(ctx.Payload.Email)
	if err != nil {
		if restErr, ok := err.(*RestClientError); ok {
			switch restErr.Code {
			case 404:
				return ctx.BadRequest(fmt.Errorf("unknown email"))
			case 400:
				return ctx.BadRequest(err)
			default:
				return ctx.InternalServerError(err)
			}
		}
		return ctx.InternalServerError(err)
	}
	// 2. Fetch user profile
	profile, err := c.fetchUserProfile(userID)
	if err != nil {
		if restErr, ok := err.(*RestClientError); ok {
			switch restErr.Code {
			case 404:
				profile = &UserProfile{
					Fullname: userID,
				}
			case 400:
				return ctx.BadRequest(err)
			default:
				return ctx.InternalServerError(err)
			}
		}
		return ctx.InternalServerError(err)
	}
	// 3. Schedule send mail
	if err = c.scheduleSendVerificationMail(userID, profile, token); err != nil {
		return ctx.InternalServerError(err)
	}

	return ctx.OK([]byte{})
}

func (c *UserController) resetVerificationToken(email string) (userID, token string, err error) {
	resetTokenPayload, err := json.Marshal(map[string]string{
		"email": email,
	})
	if err != nil {
		return "", "", err
	}
	resetTokenURL := fmt.Sprintf("%s/verification/reset", c.Config.Services["user-microservice"])
	var resetResponse *http.Response
	hystErr := hystrix.Do("user-microservice.reset_verification", func() error {
		resp, e := makeRequest(c.Client, "POST", resetTokenPayload, resetTokenURL, c.Config)
		if e != nil {
			return e
		}
		resetResponse = resp
		if resp.StatusCode != 200 {
			return extractErrorMessage(resp)
		}
		return nil
	}, nil)

	if hystErr != nil {
		return "", "", hystErr
	}

	respData, err := ioutil.ReadAll(resetResponse.Body)
	if err != nil {
		return "", "", err
	}
	tokenResponse := map[string]string{}
	if err = json.Unmarshal(respData, &tokenResponse); err != nil {
		return "", "", err
	}
	return tokenResponse["id"], tokenResponse["token"], nil
}

func (c *UserController) fetchUserProfile(userID string) (profile *UserProfile, err error) {
	fetchUserProfileURL := fmt.Sprintf("%s/%s", c.Config.Services["microservice-user-profile"], userID)
	var fetchProfileResp *http.Response
	hystErr := hystrix.Do("user-profile.get_user_profile", func() error {
		resp, e := makeRequest(c.Client, "GET", nil, fetchUserProfileURL, c.Config)
		if e != nil {
			return e
		}
		fetchProfileResp = resp
		if resp.StatusCode != 200 {
			return extractErrorMessage(resp)
		}
		return nil
	}, nil)

	if hystErr != nil {
		return nil, hystErr
	}
	bodyData, err := ioutil.ReadAll(fetchProfileResp.Body)
	if err != nil {
		return nil, err
	}
	profile = &UserProfile{}
	if err := json.Unmarshal(bodyData, &profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (c *UserController) scheduleSendVerificationMail(userID string, profile *UserProfile, token string) error {
	emailInfo := Email{
		Email: profile.Email,
		ID:    userID,
		Name:  profile.Fullname,
		Token: token,
	}

	body, err := json.Marshal(&emailInfo)
	if err != nil {
		return err
	}

	if err = c.ChannelRabbitMQ.Send("verification-email", body); err != nil {
		return err
	}

	return nil
}

func extractErrorMessage(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return &RestClientError{
			Message: "no error in response",
		}
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &RestClientError{
			Code:    -1,
			Message: fmt.Sprintf("IO Error on response read: %s", err.Error()),
		}
	}
	result := map[string]interface{}{}
	if err = json.Unmarshal(data, &result); err != nil {
		return &RestClientError{
			Code:    -1,
			Message: fmt.Sprintf("JSON Unmarshal error: %s", err.Error()),
		}
	}
	if _, ok := result["message"]; ok {
		if message, ok := result["message"].(string); ok {
			return &RestClientError{
				Code:       resp.StatusCode,
				StatusLine: resp.Status,
				Message:    message,
			}
		}
	}
	return &RestClientError{
		Code:    -1,
		Message: "Unable to get error from response. Maybe not JSON response?",
	}
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

	randUUID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	claims := jwtgo.MapClaims{
		"iss":      "microservice-registration",
		"exp":      time.Now().Add(time.Duration(30) * time.Second).Unix(),
		"jti":      randUUID.String(),
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

// RestClientError represents an error that occured in a REST call to a remote API.
type RestClientError struct {
	Code       int
	StatusLine string
	Message    string
}

func (e *RestClientError) Error() string {
	return fmt.Sprintf("%d %s %s", e.Code, e.StatusLine, e.Message)
}
