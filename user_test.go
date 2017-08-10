package main

import (
	"context"
	"fmt"
	"gopkg.in/h2non/gock.v1"
	"testing"

	"github.com/goadesign/goa"
	"github.com/JormungandrK/microservice-registration/app"
	"github.com/JormungandrK/microservice-registration/app/test"
)

var (
	service = goa.New("user-test")
	ctrl    = NewUserController(service, &MockMessage{})
)

// Call generated test helper, this checks that the returned media type is of the
// correct type (i.e. uses view "default") and validates the media type.
// Also, it ckecks the returned status code
func TestRegisterUserCreated(t *testing.T) {
	user := &app.UserPayload{
		Fullname:   "fullname",
		Username:   "username",
		Password:   "password",
		Email:      "example@mail.com",
		ExternalID: "qwerc461f9f8eb02aae053f3",
		Roles:      []string{"admin", "user"},
	}

	urlConfig, _ := UrlConfigFromFile("./urlConfig.json")

	gock.New(urlConfig.UserService).
		Post("/users/").
		Reply(201).
		JSON(map[string]interface{}{
			"id":         "59804b3c0000000000000000",
			"fullname":   user.Fullname,
			"username":   user.Username,
			"email":      user.Email,
			"externalId": "qwe04b3c000000qwertydgfsd",
			"roles":      []string{"admin", "user"},
			"active":     false,
		})

	gock.New(urlConfig.UserProfileService).
		Put(fmt.Sprintf("/users/%s/profile", "59804b3c0000000000000000")).
		Reply(204).
		JSON(map[string]interface{}{
			"fullname": user.Fullname,
			"email":    user.Email,
		})

	_, u := test.RegisterUserCreated(t, context.Background(), service, ctrl, user)

	if u == nil {
		t.Fatal("Nil user")
	}
}

// Call generated test helper, this checks that the returned media type is of the
// correct type (i.e. uses view "default") and validates the media type.
// Also, it ckecks the returned status code
func TestRegisterUserBadRequest(t *testing.T) {
	user := &app.UserPayload{
		Fullname:   "fu",
		Username:   "username",
		Password:   "password",
		Email:      "test",
		ExternalID: "qwerc461f9f8eb02aae053f3",
		Roles:      []string{"admin", "user"},
	}

	urlConfig, _ := UrlConfigFromFile("./urlConfig.json")

	gock.New(urlConfig.UserService).
		Post("/users/").
		Reply(400).
		JSON(map[string]interface{}{
			"id":         "59804b3c0000000000000000",
			"fullname":   user.Fullname,
			"username":   user.Username,
			"email":      user.Email,
			"externalId": "qwe04b3c000000qwertydgfsd",
			"roles":      []string{"admin", "user"},
			"active":     false,
		})

	gock.New(urlConfig.UserProfileService).
		Put(fmt.Sprintf("/users/%s/profile", "59804b3c0000000000000000")).
		Reply(400).
		JSON(map[string]interface{}{
			"fullname": user.Fullname,
			"email":    user.Email,
		})

	test.RegisterUserBadRequest(t, context.Background(), service, ctrl, user)
}

// func (mail *Message) SendEmail(id string, username string, email string, template string) error
func TestSendEmail(t *testing.T) {
	return nil
}

// func EmailConfigFromFile(configFile string) (*EmailConfig, error)
func TestEmailConfigFromFile(t *testing.T) {
	return nil
}

// func UrlConfigFromFile(configFile string) (*UrlConfig, error)
func TestUrlConfigFromFile(t *testing.T) {
	return nil
}