//go:generate goagen bootstrap -d microservice-registration/design

package main

import (
	"net/http"
	"os"

	"gopkg.in/gomail.v2"

	"github.com/JormungandrK/microservice-registration/app"
	"github.com/JormungandrK/microservice-registration/config"
	"github.com/JormungandrK/microservice-tools/gateway"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
)

func main() {
	// Create service
	service := goa.New("user")

	cf := os.Getenv("SERVICE_CONFIG_FILE")
	if cf == "" {
		cf = "/run/secrets/microservice_registration_config.json"
	}
	cfg, err := config.LoadConfig(cf)
	if err != nil {
		service.LogError("config", "err", err)
		return
	}

	registration := gateway.NewKongGateway(cfg.GatewayAdminURL, &http.Client{}, &cfg.Microservice)
	err = registration.SelfRegister()
	if err != nil {
		panic(err)
	}

	defer registration.Unregister() // defer the unregister for after main exits

	// Mount middleware
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(true))
	service.Use(middleware.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	// Mount "swagger" controller
	c := NewSwaggerController(service)
	app.MountSwaggerController(service, c)
	// Mount "user" controller
	c2 := NewUserController(
		service,
		&Message{msg: gomail.NewMessage()},
		cfg,
	)
	app.MountUserController(service, c2)

	// Start service
	if err := service.ListenAndServe(":8080"); err != nil {
		service.LogError("startup", "err", err)
	}
}
