Package with shared tools for the microservices
===============================================

[![Build](https://travis-ci.com/JormungandrK/microservice-tools.svg?token=UB5yzsLHNSbtjSYrGbWf&branch=master)](https://travis-ci.com/JormungandrK/microservice-tools)
[![Test Coverage](https://api.codeclimate.com/v1/badges/b2cc2c65045b0a12e6e7/test_coverage)](https://codeclimate.com/repos/59e72590aed7c5028e000970/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/b2cc2c65045b0a12e6e7/maintainability)](https://codeclimate.com/repos/59e72590aed7c5028e000970/maintainability)

This package provides tools that can be used by all microservices, such as:
 * API for service registry (Self registration and unregistration of a microservice)

# Installation

To install run:
```bash
go get -u github.com/JormungandrK/microservice-tools
```

## Install it from the private repository

You'll might get a problem when installing because the github repository is private.

By default ```go get``` will try to user a https:// to clone the repository.
If you have a valid key for accessing the repository (in ```~/.ssh/```) run the
following command to force git to use ssh:// as protocol for access and to use
your access key for the repository:

```bash
git config --global url."ssh://git@github.com:".insteadOf "https://github.com"
```

# Microservice self-registration

## Configuring the microservice

The service configuration is loaded from a JSON config file. The file has the following
structure:

```javascript
{
  "name": "user-microservice",
  "port": 8080,
  "virtual_host": "user.services.jormugandr.org",
  "hosts": ["localhost", "user.services.jormugandr.org"],
  "weight": 10,
  "slots": 100
}
```
Fields:
 * **name** - the name of the microservice. This name will be used to register the API with on Kong. This has to be the same for all instances of the microservices.
 * **port** - local port of the microservice.
 * **virtual_host** - this is the domain name for the services group. This has to be the same for all instances of the microservice.
 * **hosts** - list of valid hosts that can access the microservice. You need to put at least the **virtual_host** here, otherwise you won't be able to access the API via the gateway.
 * **weight** - an integer value used by the API gateway for load balancing.
 * **slots** - maximal number of instances (targets) for the same microservice group. When registering a service with the gateway, each instance creates a target
 on the gateway associated with the **virtual_host**. When a request passes through the gateway, the HTTP header **Host** is inspected, and based on that the **virtual_host** is determined. Then the gateway passes (proxies) the request to a specific the microservice instance. This number specifies the maximal number of targets for a group like this.


## Adding self-registration to a microservice

To add self-registration logic in your microservice, you need to create a ```gateway.Registration``` and then use ```SelfRegister()```.

Here is an example of self-registration with Kong:
```go
import (
  // other imports here

  // import the gateway package
  gateway "microservice-tools/gateway"
)

// other code in the main.go file

func main(){
  // load the Gateway URL and the config file path
  gatewayURL, configFile := loadGatewaySettings()

  // creates new Kong gateway.Registration with the config settings. We pass the default http client here.
  registration, err := gateway.NewKongGatewayFromConfigFile(gatewayURL, &http.Client{}, configFile)
  if err != nil {
    // if there is an error, it means we failed to build Registration for Kong.
    panic(err)
  }
  // at this point we do a self-registration by calling SelfRegister
  err = registration.SelfRegister()
  if err != nil {
    // if there is an error it means we failed to self-register so we panic with error
    panic(err)
  }

  // the unregistration is deferred for after main() executes. If we shut down
  // the service, it is nice to unregister, although it is not required.
  defer registration.Unregister()


  // rest of the code for main goes here
}

// loadGatewaySettings loads the API Gateway URL and the path to the JSON config file from ENV variables.
func loadGatewaySettings() (string, string) {
  gatewayURL := os.Getenv("API_GATEWAY_URL")
  serviceConfigFile := os.Getenv("SERVICE_CONFIG_FILE")

  if gatewayURL == "" {
    gatewayURL = "http://localhost:8001"
  }
  if serviceConfigFile == "" {
    serviceConfigFile = "config.json"
  }

  return gatewayURL, serviceConfigFile
}
```
