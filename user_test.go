package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"gopkg.in/h2non/gock.v1"

	"github.com/JormungandrK/microservice-registration/app"
	"github.com/JormungandrK/microservice-registration/app/test"
	"github.com/JormungandrK/microservice-registration/config"
	"github.com/goadesign/goa"
)

var configBytes = []byte(`
		{
			"microservice": {
				"name": "registration-microservice",
				"port": 8080,
				"paths": ["/users/register"],
				"virtual_host": "registration.services.jormugandr.org",
				"weight": 10,
				"slots": 100
		    },
			"gatewayUrl": "http://kong:8000",
		  	"gatewayAdminUrl": "http://kong:8001",
		    "services": {
		      "user-microservice": "http://kong:8000/users",
		      "microservice-user-profile": "http://kong:8000/profiles"
		    },
		    "mail": {
				"host": "smtp.gmail.com",
				"port": "587",
				"user": "user_email",
				"password": "password"
		    }
		}
	`)
var cfg = &config.Config{}
var _ = json.Unmarshal(configBytes, cfg)

var (
	service = goa.New("user-test")
	ctrl    = NewUserController(service, &MockMessage{}, cfg)
)

// Call generated test helper, this checks that the returned media type is of the
// correct type (i.e. uses view "default") and validates the media type.
// Also, it ckecks the returned status code
func TestRegisterUserCreated(t *testing.T) {
	pass := "password"
	extID := "qwerc461f9f8eb02aae053f3"
	user := &app.UserPayload{
		Fullname:   "fullname",
		Username:   "username",
		Password:   &pass,
		Email:      "example@mail.com",
		ExternalID: &extID,
		Roles:      []string{"admin", "user"},
	}

	gock.New(cfg.Services["user-microservice"]).
		Post("").
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

	gock.New(cfg.Services["microservice-user-profile"]).
		Put(fmt.Sprintf("/%s", "59804b3c0000000000000000")).
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
	pass := "password"
	extID := "qwerc461f9f8eb02aae053f3"
	user := &app.UserPayload{
		Fullname:   "fu",
		Username:   "username",
		Password:   &pass,
		Email:      "test",
		ExternalID: &extID,
		Roles:      []string{"admin", "user"},
	}

	gock.New(cfg.Services["user-microservice"]).
		Post("").
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

	gock.New(cfg.Services["microservice-user-profile"]).
		Put(fmt.Sprintf("/%s", "59804b3c0000000000000000")).
		Reply(400).
		JSON(map[string]interface{}{
			"fullname": user.Fullname,
			"email":    user.Email,
		})

	test.RegisterUserBadRequest(t, context.Background(), service, ctrl, user)
}
