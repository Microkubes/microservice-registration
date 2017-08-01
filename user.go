package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/goadesign/goa"
	"gopkg.in/gomail.v2"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"userRegistration-microservice/app"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
}

// EmailConfig represents configuration for email.
type EmailConfig struct {
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

// Represents urls from the external services
type UrlConfig struct {
	UserService        string `json:"userService,omitempty"`
	UserProfileService string `json:"userProfileService,omitempty"`
}

// Represents User Profle
type UserProfile struct {
	Fullname string
	Email    string
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service) *UserController {
	return &UserController{Controller: service.NewController("UserController")}
}

// Register runs the register action.
func (c *UserController) Register(ctx *app.RegisterUserContext) error {
	user := &app.Users{}
	client := &http.Client{}
	urlConfig, errUrl := UrlConfigFromFile("./urlConfig.json")
	if errUrl != nil {
		return errUrl
	}

	// Create new user from payload
	jsonUser, error := json.Marshal(ctx.Payload)
	if error != nil {
		return error
	}
	resp, err := CreateNewUser(client, jsonUser, urlConfig.UserService)
	if err != nil {
		return err
	}

	// Inspect status code from response
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		// Temporary workaround.
		response := strings.Replace(string(body), "\"", "'", -1)
		response = strings.Replace(response, "\\", "", -1)

		err = errors.New(response)
		return ctx.BadRequest(goa.ErrBadRequest(err))
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
	resp, errUserProfile := UpdateUserProfile(client, jsonUseProfile, user.ID, urlConfig.UserProfileService)
	if errUserProfile != nil {
		return errUserProfile
	}

	// Inspect status code from response
	bodyUserProfile, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		// Temporary workaround.
		response := strings.Replace(string(bodyUserProfile), "\"", "'", -1)
		response = strings.Replace(response, "\\", "", -1)

		err = errors.New(response)
		return ctx.BadRequest(goa.ErrBadRequest(err))
	}

	// Send email for verification
	if err = SendEmail(user.ID, user.Fullname, user.Email); err != nil {
		return err
	}

	return ctx.Created(user)
}

// Create new user.
func CreateNewUser(client *http.Client, payload []byte, url string) (*http.Response, error) {
	resp, err := client.Post(fmt.Sprintf("%s/users/", url), "application/json", bytes.NewBuffer(payload))
	return resp, err
}

// Update user profile.
func UpdateUserProfile(client *http.Client, payload []byte, id string, url string) (*http.Response, error) {
	resp, err := PutRequest(fmt.Sprintf("%s/users/%s/profile", url, id), bytes.NewBuffer(payload), client)
	return resp, err
}

// Send email for verification.
func SendEmail(id string, username string, email string) error {
	emailConfig, _ := EmailConfigFromFile("./emailConfig.json")

	m := gomail.NewMessage()
	m.SetHeader("From", emailConfig.User)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Verify Your Account!")
	m.SetBody("text/html", fmt.Sprintf("Hello %s, <p><a href=\"http://localhost:8081/users/%s/verify\">Verify your account</a></p>", username, id))

	d := gomail.NewDialer(emailConfig.Host, emailConfig.Port, emailConfig.User, emailConfig.Password)

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

// Read email configuratio from config file.
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

// Read url configuratio from config file.
func UrlConfigFromFile(configFile string) (*UrlConfig, error) {
	var config UrlConfig
	cnf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(cnf, &config)
	if err != nil {
		return nil, err
	}

	// Validate urls
	c := &config
	_, errUrlUser := url.ParseRequestURI(string(c.UserService))
	if errUrlUser != nil {
		return nil, errUrlUser
	}
	_, errUrlUserProfile := url.ParseRequestURI(string(c.UserService))
	if errUrlUserProfile != nil {
		return nil, errUrlUserProfile
	}

	return &config, nil
}

// Because http.Client does not provide PUT method
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
