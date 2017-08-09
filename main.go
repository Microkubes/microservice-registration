//go:generate goagen bootstrap -d microservice-registration/design

package main

import (
	"gopkg.in/gomail.v2"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	"github.com/JormungandrK/microservice-registration/app"
)

func main() {
	// Create service
	service := goa.New("user")

	// Mount middleware
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(true))
	service.Use(middleware.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	// Mount "swagger" controller
	c := NewSwaggerController(service)
	app.MountSwaggerController(service, c)
	// Mount "user" controller
	c2 := NewUserController(service, &Message{msg: gomail.NewMessage()})
	app.MountUserController(service, c2)

	// Start service
	if err := service.ListenAndServe(":8081"); err != nil {
		service.LogError("startup", "err", err)
	}


}
