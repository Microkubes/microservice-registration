package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"gopkg.in/h2non/gock.v1"

	"github.com/JormungandrK/microservice-registration/app"
	"github.com/JormungandrK/microservice-registration/app/test"
	"github.com/JormungandrK/microservice-registration/config"
	"github.com/JormungandrK/microservice-tools/rabbitmq"
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
		  	"systemKey": "system",
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
	ctrl    = NewUserController(service, cfg, &rabbitmq.MockAMQPChannel{}, &http.Client{})
)

func TestMain(m *testing.M) {
	privkey, _ := rsa.GenerateKey(rand.Reader, 2048)
	bytes := x509.MarshalPKCS1PrivateKey(privkey)
	privateBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bytes,
	})
	ioutil.WriteFile("system", privateBytes, 0644)
	defer os.Remove("system")
	m.Run()
}

// Call generated test helper, this checks that the returned media type is of the
// correct type (i.e. uses view "default") and validates the media type.
// Also, it ckecks the returned status code
func TestRegisterUserCreated(t *testing.T) {
	pass := "password"
	extID := "qwerc461f9f8eb02aae053f3"
	user := &app.UserPayload{
		Fullname:   "fullname",
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

	gock.InterceptClient(ctrl.Client)
	_, u := test.RegisterUserCreated(t, context.Background(), service, ctrl, user)

	if u == nil {
		t.Fatal("Nil user")
	}
}

// TestRegisterUserInternalServerError tests internal server error scenario
func TestRegisterUserInternalServerError(t *testing.T) {
	pass := "password"
	extID := "qwerc461f9f8eb02aae053f3"
	user := &app.UserPayload{
		Fullname:   "fu",
		Password:   &pass,
		Email:      "test",
		ExternalID: &extID,
		Roles:      []string{"admin", "user"},
	}

	gock.New(cfg.Services["user-microservice"]).
		Post("").
		Reply(500).
		JSON(map[string]interface{}{
			"id":         "59804b3c0000000000000000",
			"fullname":   user.Fullname,
			"email":      user.Email,
			"externalId": "qwe04b3c000000qwertydgfsd",
			"roles":      []string{"admin", "user"},
			"active":     false,
		})

	gock.New(cfg.Services["microservice-user-profile"]).
		Put(fmt.Sprintf("/%s", "59804b3c0000000000000000")).
		Reply(500).
		JSON(map[string]interface{}{
			"fullname": user.Fullname,
			"email":    user.Email,
		})
	gock.InterceptClient(ctrl.Client)
	test.RegisterUserInternalServerError(t, context.Background(), service, ctrl, user)
}

// Call generated test helper, this checks that the returned media type is of the
// correct type (i.e. uses view "default") and validates the media type.
// Also, it ckecks the returned status code
func TestRegisterUserBadRequest(t *testing.T) {
	pass := "password"
	extID := "qwerc461f9f8eb02aae053f3"
	user := &app.UserPayload{
		Fullname:   "fu",
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
	gock.InterceptClient(ctrl.Client)
	test.RegisterUserBadRequest(t, context.Background(), service, ctrl, user)
}

func TestMakeRequest(t *testing.T) {

	payload := []byte(`{
	    "data": "something"
	  }`)
	client := &http.Client{}

	gock.New("http://test.com").
		Post("/users").
		Reply(201)

	gock.InterceptClient(client)
	resp, err := makeRequest(client, http.MethodPost, payload, "http://test.com/users", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("Nil response")
	}
}

func TestSelfSignJWT(t *testing.T) {

	token, err := selfSignJWT(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if token == "" {
		t.Fatal("Empty JWT token")
	}
}

func TestResenVerification_OK(t *testing.T) {
	gock.Off()

	gock.New("http://kong:8000").
		Post("/users/verification/reset").
		JSON(map[string]interface{}{
			"email": "email@example.com",
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"id":    "user-id",
			"email": "email@example.com",
			"token": "verification_token_reset",
		})

	gock.New("http://kong:8000").
		Get("/profiles/user-id").
		Reply(200).JSON(map[string]interface{}{
		"userId":   "user-id",
		"email":    "email@example.com",
		"fullName": "Test User",
	})

	gock.InterceptClient(ctrl.Client)
	payload := &app.ResendVerificationPayload{
		Email: "email@example.com",
	}
	test.ResendVerificationUserOK(t, context.Background(), service, ctrl, payload)
}

func TestResendVerification_BadRequest(t *testing.T) {
	gock.Off()
	gock.New("http://kong:8000").
		Post("/users/verification/reset").
		JSON(map[string]interface{}{
			"email": "email@example.com",
		}).
		Reply(400).
		JSON(map[string]interface{}{
			"id":      "XYZ_REQ_ID",
			"message": "already activated",
		})

	gock.InterceptClient(ctrl.Client)

	test.ResendVerificationUserBadRequest(t, context.Background(), service, ctrl, &app.ResendVerificationPayload{
		Email: "email@example.com",
	})
}

func TestResenVerification_InternalServerError(t *testing.T) {
	gock.Off()

	gock.New("http://kong:8000").
		Post("/users/verification/reset").
		JSON(map[string]interface{}{
			"email": "email@example.com",
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"id":    "user-id",
			"email": "email@example.com",
			"token": "verification_token_reset",
		})

	gock.New("http://kong:8000").
		Get("/profiles/user-id").
		Reply(500).
		JSON(map[string]interface{}{
			"id":      "XYZ_2_REQ_ID",
			"message": "unknown internal error",
		})
	gock.InterceptClient(ctrl.Client)

	test.ResendVerificationUserInternalServerError(t, context.Background(), service, ctrl, &app.ResendVerificationPayload{
		Email: "email@example.com",
	})
}
