User Registration Microservice
==============================

[![Build](https://travis-ci.com/Microkubes/microservice-registration.svg?token=UB5yzsLHNSbtjSYrGbWf&branch=master)](https://travis-ci.com/Microkubes/microservice-registration)
[![Test Coverage](https://api.codeclimate.com/v1/badges/a30f92c1b70b692e6484/test_coverage)](https://codeclimate.com/repos/59e7705a9c7963028d001870/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/a30f92c1b70b692e6484/maintainability)](https://codeclimate.com/repos/59e7705a9c7963028d001870/maintainability)

## Prerequisite
Create a project directory. Set GOPATH enviroment variable to that project. Add $GOPATH/bin to the $PATH
```
export GOPATH=/path/to/project-workspace
export PATH=$GOPATH/bin:$PATH
```
Install goa and goagen:
```
cd $GOPATH
go get -u github.com/goadesign/goa/...
```

## Compile and run the service:
Clone the repo:
```
cd $GOPATH/src
git clone https://github.com/Microkubes/microservice-registration.git /path/to/project-workspace/src/github.com/Microkubes/microservice-registration
```
Be sure to use the full domain name and resource path here (compatible with ```go get```).


Then compile and run:
```
cd /path/to/project-workspace/src/github.com/Microkubes/microservice-registration
go build -o registration
./registration
```

## Change the design
If you change the design then you should regenerate the files. Run:
```
cd /path/to/project-workspace/src/github.com/Microkubes/microservice-registration
go generate
```
**NOTE:** If the above command does not update the generated code per the changes in the design,
then run ```goagen bootstrap```:

```bash
goagen bootstrap -d github.com/Microkubes/microservice-registration/design -o .
```


Also, recompile the service and start it again:
```
go build -o registration
./registration
```

## Other changes, not related to the design
For all other changes that are not related to the design just recompile the service and start it again:
```
cd $GOPATH/src/github.com/Microkubes/microservice-registration
go build -o registration
./registration
```

## Tests
For testing we use controller_test.go files which call the generated test helpers which package that data into HTTP requests and calls the actual controller functions. The test helpers retrieve the written responses, deserialize them, validate the generated data structures (against the validations written in the design) and make them available to the tests. Run:
```
go test -v
```

# Docker Builds

First, create a directory for the shh keys:
```bash
mkdir keys
```

Find a key that you'll use to acceess Microkubes organization on github. Then copy the
private key to the directory you created above. The build would use this key to
access ```Microkubes/microservice-tools``` repository.

```bash
cp ~/.ssh/id_rsa keys/
```

**WARNING!** Make sure you don't commit or push this key to the repository!

To build the docker image of the microservice, run the following command:
```bash
docker build -t microservice-registration .
```

Also, you can build docker image using Makefile. Run the following command:
```bash
make build
```

# Running the microservice

To run the registration microservice you'll need to set up some ENV variables:

 * **SERVICE_CONFIG_FILE** - Location of the configuration JSON file

Run the docker image:
```bash
docker run microservice-registration
```

## Check if the service is self-registering on Kong Gateway

First make sure you have started Kong. See [Jormungandr Infrastructure](https://github.com/Microkubes/jormungandr-infrastructure)
on how to set up Kong locally.

If you have Kong admin endpoint running on http://localhost:8001 , you're good to go.
Build and run the service:
```bash
go build -o registration
./registration
```

To access the registration service, then instead of calling the service on :8081 port,
make the call to Kong:

```bash
curl -v -X POST http://localhost:8000/users/profiles
```

You should see a log on the terminal running the service that it received and handled the request.

## Running with the docker image

Assuming that you have Kong and it is availabel od your host (ports: 8001 - admin, and 8000 - proxy) and
you have build the service docker image (microservice-registration), then you need to pass
the Kong URL as an ENV variable to the docker run. This is needed because by default
the service will try http://localhost:8001 inside the container and won't be able to connect to kong.

Find your host IP using ```ifconfig``` or ```ip addr```.
Assuming your host IP is 192.168.1.10, then run:

```bash
docker run -ti microservice-registration
```

Also, you can build and run docker image using Makefile. Run:
```bash
make run
```

If there are no errors, on a different terminal try calling Kong on port :8000

```bash
curl -v -X POST http://localhost:8000/users/profiles
```

You should see output (log) in the container running the service.


# Service configuration

The service loads the gateway configuration from a JSON file /run/secrets/microservice_registration_config.json. To change the path set the
**SERVICE_CONFIG_FILE** env var.
Here's an example of a JSON configuration file:

```json
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
	"systemKey": "/run/secrets/system",
	"verificationURL": "http://kong:8000/users",
	"services": {
		"user-microservice": "http://kong:8000/users",
		"microservice-user-profile": "http://kong:8000/profiles"
	},
	"mail": {
		"host": "smtp.example.com",
		"port": "587",
		"user": "user-mail",
		"password": "password"
	},
	"rabbitmq": {
		"username": "guest",
		"password": "guest",
		"host": "rabbitmq",
		"port": "5672"
	}
}
```

Configuration properties:
 * **name** - ```"registration-microservice"``` - the name of the service, do not change this.
 * **port** - ```8080``` - port on which the microservice is running.
 * **paths** - microservice base paths
 * **virtual_host** - ```"registration.services.jormugandr.org"``` domain name of the service group/cluster. Don't change if not sure.
 * **weight** - instance weight - user for load balancing.
 * **slots** - maximal number of service instances under ```"registration.services.jormugandr.org"```.
 * **gatewayUrl** -  kong proxy url
 * **gatewayAdminUrl** -  kong admin url
 * **systemKey** -  path to rhe system key. On docker swarm it should be /run/secrets/system
 * **verificationURL** -  client verification url (format <url>/userID/verify )
 * **services** - holds the urls of the microservices
 * **mail** - holds mail settings
 * **rabbitmq** - holds info about RabbitMQ server

 ## Contributing

For contributing to this repository or its documentation, see the [Contributing guidelines](CONTRIBUTING.md).