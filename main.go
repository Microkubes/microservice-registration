//go:generate goagen bootstrap -d microservice-registration/design

package main

import (
	"net/http"
	"os"

	"github.com/Microkubes/microservice-registration/app"
	"github.com/Microkubes/microservice-registration/config"
	"github.com/Microkubes/microservice-tools/gateway"
	"github.com/Microkubes/microservice-tools/rabbitmq"
	"github.com/Microkubes/microservice-tools/utils/healthcheck"
	"github.com/Microkubes/microservice-tools/utils/version"
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
		panic(err)
	}

	connRabbitMQ, channelRabbitMQ, err := rabbitmq.Dial(
		cfg.RabbitMQ["username"],
		cfg.RabbitMQ["password"],
		cfg.RabbitMQ["host"],
		cfg.RabbitMQ["post"],
	)
	if err != nil {
		service.LogError("rabbitmq", "err", err)
		panic(err)
	}
	defer connRabbitMQ.Close()
	defer channelRabbitMQ.Close()

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

	service.Use(healthcheck.NewCheckMiddleware("/healthcheck"))

	service.Use(version.NewVersionMiddleware(cfg.Version, "/version"))

	// Mount "swagger" controller
	c := NewSwaggerController(service)
	app.MountSwaggerController(service, c)
	// Mount "user" controller
	c2 := NewUserController(
		service,
		cfg,
		&rabbitmq.AMQPChannel{
			Channel: channelRabbitMQ,
		},
		&http.Client{},
	)
	app.MountUserController(service, c2)

	// Start service
	if err := service.ListenAndServe(":8080"); err != nil {
		service.LogError("startup", "err", err)
	}
}
