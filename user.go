package main

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"errors"
	"strings"
	// "log"
	"fmt"
	"gopkg.in/gomail.v2"
	"github.com/goadesign/goa"
	"userRegistration-microservice/app"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
}

// EmailConfig represents configuration for email.
type EmailConfig struct {
	Host string `json:"host,omitempty"`
	Port int `json:"port,omitempty"`
	User string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service) *UserController {
	return &UserController{Controller: service.NewController("UserController")}
}

// Register runs the register action.
func (c *UserController) Register(ctx *app.RegisterUserContext) error {
	var user *app.Users
	client := &http.Client{}

	jsonPayload, error := json.Marshal(ctx.Payload)
	if error != nil {
		return error
	}

	resp, err := CreateNewUser(client, jsonPayload)
	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		// Temporary workaround.
		response := strings.Replace(string(body), "\"", "'", -1)
		response = strings.Replace(response, "\\", "", -1)
		
		err = errors.New(response)
		ctx.BadRequest(goa.ErrBadRequest(err))
	}

	// Unmarshal response from create user service
	if err = json.Unmarshal(body, &user); err != nil {
		return err
	}

	// Send mail.
	if err = SendEmail(user.ID, ctx.Payload.Fullname, user.Email); err != nil {
		return err
	}

	return nil
}

func CreateNewUser(client *http.Client, payload []byte) (*http.Response, error) {
	resp, err := client.Post("http://127.0.0.1:8080/users/", "application/json", bytes.NewBuffer(payload))
	return resp, err
}

func SendEmail(id string, username string, email string) error {
	// Load email settings.
	emailConfig, _ := EmailConfigFromFile("./emailConfig.json")

	m := gomail.NewMessage()
	m.SetHeader("From", emailConfig.User)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Verify Your Account!")
	m.SetBody("text/html", fmt.Sprintf("Hello %s, <p><a href=\"http://localhost:8081/users/verify/%s\">Verify your account</a></p>", username, id))

	d := gomail.NewDialer(emailConfig.Host, emailConfig.Port, emailConfig.User, emailConfig.Password)

	// Send the email.
	if err := d.DialAndSend(m); err != nil {
	    panic(err)
	}	
	return nil
}

func EmailConfigFromFile(configFile string) (*EmailConfig, error) {
	var config EmailConfig
	cnf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(cnf, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}