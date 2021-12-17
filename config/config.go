package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/Microkubes/microservice-tools/gateway"

	commonconf "github.com/Microkubes/microservice-tools/config"
)

// Config holds the microservice full configuration.
type Config struct {
	// Microservice is a gateway.Microservice configuration for self-registration and service config.
	Microservice gateway.MicroserviceConfig `json:"microservice"`

	// Database holds the database configuration
	Database *commonconf.DBConfig `json:"database"`

	// SystemKey holds the path to the system key which is provate RSA key
	SystemKey string `json:"systemKey"`

	// Services is a map of <service-name>:<service base URL>. For example,w
	// "user-microservice": "http://kong.gateway:8001/user"
	Services map[string]string `json:"services"`

	// Mail is a map of <property>:<value>. For example,
	// "host": "smtp.example.com"
	Mail map[string]string `json:"mail"`

	// RabbitMQ holds information about the rabbitmq server
	RabbitMQ map[string]string `json:"rabbitmq"`

	//Version is version of the service
	Version string `json:"version"`
}

// LoadConfig loads a Config from a configuration JSON file.
func LoadConfig(confFile string) (*Config, error) {
	confBytes, err := ioutil.ReadFile(confFile)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = json.Unmarshal(confBytes, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
