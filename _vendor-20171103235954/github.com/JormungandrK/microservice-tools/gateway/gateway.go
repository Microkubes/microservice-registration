package gateway

// Registration registers and unregisters microservices on the API Gateway.
type Registration interface {

	// SelfRegister performs a self registration of the microservice against an API Gateway.
	SelfRegister() error

	// Unregister unregisters previously registerd microservice on an API Gateway.
	Unregister() error
}
