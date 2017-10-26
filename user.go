package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	gomail "gopkg.in/gomail.v2"

	"github.com/JormungandrK/microservice-registration/app"
	"github.com/JormungandrK/microservice-registration/config"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/goadesign/goa"
)

// CollectionEmail is an interface to access to the email func.
type CollectionEmail interface {
	SendEmail(id string, username string, email string, template string, cfg *config.Config) error
}

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	emailCollection CollectionEmail
	Config          *config.Config
}

// Message wraps a gomail.Message to embed methods in models.
type Message struct {
	msg *gomail.Message
}

// MockMessage for testing
type MockMessage struct{}

// For the email template
type email struct {
	ID   string
	Name string
}

// UserProfile represents User Profle
type UserProfile struct {
	Fullname string
	Email    string
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, emailCollection CollectionEmail, config *config.Config) *UserController {
	return &UserController{
		Controller:      service.NewController("UserController"),
		emailCollection: emailCollection,
		Config:          config,
	}
}

// Register runs the register action. It creates a user and user profile.
// Also, it sends an email to the user if user does not come from external services.
func (c *UserController) Register(ctx *app.RegisterUserContext) error {
	user := &app.Users{}
	client := &http.Client{}

	// Create new user from payload
	jsonUser, errJSONUser := json.Marshal(ctx.Payload)
	if errJSONUser != nil {
		return errJSONUser
	}

	output := make(chan *http.Response, 1)
	errorsChan := hystrix.Go("user-microservice.create_user", func() error {
		resp, err := CreateNewUser(client, jsonUser, c.Config.Services["user-microservice"])
		if err != nil {
			return err
		}
		output <- resp
		return nil
	}, nil)

	var createUserResp *http.Response
	var err error
	select {
	case out := <-output:
		createUserResp = out
	case respErr := <-errorsChan:
		return respErr
	}

	// Inspect status code from response
	body, _ := ioutil.ReadAll(createUserResp.Body)
	if createUserResp.StatusCode != 200 && createUserResp.StatusCode != 201 {
		// Temporary workaround.
		response := strings.Replace(string(body), "\"", "'", -1)
		response = strings.Replace(response, "\\", "", -1)

		err = errors.New(response)
		return err
	}

	if err = json.Unmarshal(body, &user); err != nil {
		return err
	}

	user.Fullname = ctx.Payload.Fullname

	// Update user profile. Create it if does not exist
	userProfile := UserProfile{user.Fullname, user.Email}
	jsonUseProfile, errMarshalUserProfile := json.Marshal(userProfile)
	if errMarshalUserProfile != nil {
		return errMarshalUserProfile
	}

	upOutput := make(chan *http.Response, 1)

	upErrorChan := hystrix.Go("user-microservice.update_user_profile", func() error {
		resp, errUserProfile := UpdateUserProfile(client, jsonUseProfile, user.ID, c.Config.Services["microservice-user-profile"])
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
		return respErr
	}

	// Inspect status code from response
	bodyUserProfile, _ := ioutil.ReadAll(createUpResp.Body)
	if createUpResp.StatusCode != 200 && createUpResp.StatusCode != 204 {
		// Temporary workaround.
		response := strings.Replace(string(bodyUserProfile), "\"", "'", -1)
		response = strings.Replace(response, "\\", "", -1)

		err = errors.New(response)
		return ctx.BadRequest(goa.ErrBadRequest(err))
	}

	if ctx.Payload.ExternalID == nil {
		userEmail := email{user.ID, user.Fullname}

		template, errTemp := ParseTemplate("./emailTemplate.html", userEmail)
		if errTemp != nil {
			return errTemp
		}

		// Send email for verification
		if err = c.emailCollection.SendEmail(user.ID, user.Fullname, user.Email, template, c.Config); err != nil {
			return err
		}
	}

	return ctx.Created(user)
}

// CreateNewUser creates a new user.
func CreateNewUser(client *http.Client, payload []byte, url string) (*http.Response, error) {
	resp, err := client.Post(fmt.Sprintf("%s/users", url), "application/json", bytes.NewBuffer(payload))
	return resp, err
}

// UpdateUserProfile updates user profile.
func UpdateUserProfile(client *http.Client, payload []byte, id string, url string) (*http.Response, error) {
	resp, err := PutRequest(fmt.Sprintf("%s/profiles/%s", url, id), bytes.NewBuffer(payload), client)
	return resp, err
}

// SendEmail sends an email for verification.
func (mail *Message) SendEmail(id string, username string, email string, template string, cfg *config.Config) error {
	mail.msg.SetHeader("From", cfg.Mail["user"])
	mail.msg.SetHeader("To", email)
	mail.msg.SetHeader("Subject", "Verify Your Account!")
	mail.msg.SetBody("text/html", template)

	port, err := strconv.Atoi(cfg.Mail["port"])
	if err != nil {
		return err
	}
	d := gomail.NewDialer(cfg.Mail["host"], port, cfg.Mail["user"], cfg.Mail["password"])

	if err := d.DialAndSend(mail.msg); err != nil {
		return err
	}
	return nil
}

// SendEmail mock sends email for verification.
func (mail *MockMessage) SendEmail(id string, username string, email string, template string) error {
	return nil
}

// ParseTemplate creates a template using emailTemplate.html
func ParseTemplate(templateFileName string, data interface{}) (string, error) {
	tmpl, err := template.ParseFiles(templateFileName)
	if err != nil {
		return "", err
	}

	// Stores the parsed template
	var buff bytes.Buffer

	// Send the parsed template to buff
	err = tmpl.Execute(&buff, data)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}

// PutRequest Because http.Client does not provide PUT method
func PutRequest(url string, data io.Reader, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, url, data)
	if err != nil {
		return nil, err
	}
	resp, error := client.Do(req)
	if error != nil {
		return resp, error
	}

	return resp, nil
}
