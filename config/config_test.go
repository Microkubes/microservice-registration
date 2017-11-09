package config

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	config := `{
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
	  	"systemKey": "/run/secrets/system",
	    "services": {
	      "user-microservice": "http://127.0.0.1:8080",
	      "microservice-user-profile": "http://127.0.0.1:8082"
	    },
	    "mail": {
			"host": "smtp.example.com",
			"port": "587",
			"user": "user_email",
			"password": "password"
	    },
		"rabbitmq": {
			"username": "guest",
			"password": "guest",
			"host": "rabbitmq",
			"port": "5672"
		}
	  }`

	cnfFile, err := ioutil.TempFile("", "tmp-config")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(cnfFile.Name())

	cnfFile.WriteString(config)

	cnfFile.Sync()

	loadedCnf, err := LoadConfig(cnfFile.Name())

	if err != nil {
		t.Fatal(err)
	}

	if loadedCnf == nil {
		t.Fatal("Configuration was not read")
	}
}
