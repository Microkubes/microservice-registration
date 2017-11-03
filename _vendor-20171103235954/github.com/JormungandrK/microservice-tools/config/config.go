package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/JormungandrK/microservice-tools/gateway"
)

// ServiceConfig holds the full microservice configuration:
// - Configuration for registering on the API Gateway
// - Security configuration
// - Database configuration
type ServiceConfig struct {
	// Service holds the confgiuration for connecting and registering the service with the API Gateway
	Service *gateway.MicroserviceConfig `json:"service"`
	// SecurityConfig holds the security configuration
	SecurityConfig `json:"security, omitempty"`
	// DBConfig holds the database connection configuration
	DBConfig `json:"database"`
	// GatewayURL is the URL of the API Gateway
	GatewayURL string `json:"gatewayUrl"`
	// GatewayAdminURL is the administration URL of the API Gateway. Used for purposes of registration of a
	// microservice with the API gateway.
	GatewayAdminURL string `json:"gatewayAdminUrl"`
}

// DBConfig holds the database configuration parameters.
type DBConfig struct {

	// Host is the database host+port URL
	Host string `json:"host,omitempty"`

	// Username is the username used to access the database
	Username string `json:"user,omitempty"`

	// Password is the databse user password
	Password string `json:"pass,omitempty"`

	// DatabaseName is the name of the database where the server will store the collections
	DatabaseName string `json:"database,omitempty"`
}

// LoadConfig loads the service configuration from a file.
func LoadConfig(confFile string) (*ServiceConfig, error) {
	data, err := ioutil.ReadFile(confFile)
	if err != nil {
		return nil, err
	}
	conf := &ServiceConfig{}
	err = json.Unmarshal(data, conf)
	return conf, err
}
