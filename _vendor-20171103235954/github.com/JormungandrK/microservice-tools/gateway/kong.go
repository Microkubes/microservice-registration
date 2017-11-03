package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// KongGateway holds the configuration and values for a pre-defined Kong API Gateway.
type KongGateway struct {
	// GatewayURL is the admin URL of the kong gateway. This is usually the URL (host plus port) of Kong admin
	GatewayURL string
	config     *MicroserviceConfig
	client     *http.Client
}

// MicroserviceConfig represents configuration for the microservice itself.
type MicroserviceConfig struct {

	// MicroserviceName is the name of the microservice. This microservice will be registered on the Gateway under this name.
	// Note that this is not the domain (host) of the microservice, but a human readable name of the microservice.
	MicroserviceName string `json:"name,omitempty"`

	// MicroservicePort is the local port on which the microservice is exposed.
	MicroservicePort int `json:"port,omitempty"`

	// VirtualHost is the domain name of the virtual host for all microservices of this name.
	// We can have multiple instances (containers) running on a single platform. Every microservice instance must have the same
	// virtual host name to be a part of the same cluster. For example, if you are exposing a user microservices, then a
	// virtual host might be 'user.services.mydomain.com'. When accessing user microservices you'll call http://user.services.mydomain.com:8000/user
	// which will redirect to a specific microservice.
	// This is a configuration for 'upstream' in Kong Gateway.
	VirtualHost string `json:"virtual_host,omitempty"`

	Paths []string `json:"paths,omitempty"`

	// Hosts is a list of supported hosts by the microservice.
	// When accessing the microservice, you must set the HTTP header 'Host' to a value
	// that is listed in this list of hosts.
	Hosts []string `json:"hosts,omitempty"`

	// Weight is the weight of this particular microservice used for load ballancing by the gateway.
	Weight int `json:"weight,omitempty"`

	// ServicesMaxSlots is the maximal number of slots which the load ballancer on the gateway will
	// allocate for the this VirtualHost.
	ServicesMaxSlots int `json:"slots,omitempty"`
}

// NewKongGateway creates a Kong Gateway with the given admin URL of kong, an http.Client and a MicroserviceConfig.
func NewKongGateway(adminURL string, client *http.Client, config *MicroserviceConfig) *KongGateway {
	return &KongGateway{
		GatewayURL: adminURL,
		config:     config,
		client:     client,
	}
}

// NewKongGatewayFromConfigFile creates a Kong Gateway for a given admin URL of kong, an http.Client and
// a location of a JSON file with the configuration.
// The configuration JSON has the following structure:
// 	{
//		"name": "The name of the service",
//		"port": 8080, // the local microservice port
//		"virtual_host": "Microservices upstream virtual host",
// 		"hosts": ["localhost", "example.org"] // valid HTTP Host header values for this microservice
// 		"weight": 10, // microservice instance weight used for load ballancing
// 		"slots": 100 // maximal number of slots to allocate for this microservices group
// }
func NewKongGatewayFromConfigFile(adminURL string, client *http.Client, configFile string) (*KongGateway, error) {
	var config MicroserviceConfig
	cnf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(cnf, &config)
	if err != nil {
		return nil, err
	}

	return NewKongGateway(adminURL, client, &config), nil
}

// SelfRegister performs a self registration of the microservice with a Kong Gateway.
// It performs the following tasks:
// 1. Checks for existence of 'upstream' for the microservices group. If there is no 'upstream'
// configured, it creates a new one with the given configuration.
// 2. Checks for the existince of API for the microservices group. If there is no API
// created on Kong, it creates a new one with the given configuration.
// 3. Adds new target on kong for the configured 'upstream' and 'API'.
func (kong *KongGateway) SelfRegister() error {
	err := kong.createOrUpdateUpstream(kong.config.VirtualHost, kong.config.ServicesMaxSlots)
	if err != nil {
		return err
	}

	apiConf := NewAPIConf()
	// TODO: map here from config
	apiConf.Name = kong.config.MicroserviceName
	apiConf.Hosts = kong.config.Hosts
	apiConf.UpstreamURL = fmt.Sprintf("http://%s:%d", kong.config.VirtualHost, kong.config.MicroservicePort)
	apiConf.URIs = kong.config.Paths
	_, err = kong.createOrUpdateAPI(apiConf)
	if err != nil {
		return err
	}

	_, err = kong.addSelfAsTarget(kong.config.VirtualHost, kong.config.MicroservicePort, kong.config.Weight)
	return err
}

// Unregister unregisters this instance of the microservice from the Kong Gateway.
// Basically it updates the upstream target with weight 0, which disables the target.
func (kong *KongGateway) Unregister() error {
	_, err := kong.addSelfAsTarget(kong.config.VirtualHost, kong.config.MicroservicePort, 0)
	return err
}

// upstream is internally used structure that represents Kong's 'upstream' object.
// See https://getkong.org/docs/0.10.x/admin-api/#upstream-object
type upstream struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	OrderList []int  `json:"orderlist,omitempty"`
	Slots     int    `json:"slots,omitempty"`
	CreatedAt int    `json:"created_at,omitempty"`
}

// upstreamTarget is internally used structure that represents Kong's 'upstream-target' object.
// See https://getkong.org/docs/0.10.x/admin-api/#target-object
type upstreamTarget struct {
	ID         string `json:"id,omitempty"`
	Target     string `json:"target,omitempty"`
	Weight     int    `json:"weight,omitempty"`
	UpstreamID string `json:"upstream_id,omitempty"`
	CreatedAt  int    `json:"created_at,omitempty"`
}

// API is a structure that represents Kong's API object.
// See https://getkong.org/docs/0.10.x/admin-api/#api-object
type API struct {
	ID                     string   `json:"id,omitempty"`
	CreatedAt              int      `json:"created_at,omitempty"`
	Hosts                  []string `json:"hosts,omitempty"`
	URIs                   []string `json:"-"`
	Methods                []string `json:"-"`
	HTTPIfTerminated       bool     `json:"http_if_terminated,omitempty"`
	HTTPSOnly              bool     `json:"https_only,omitempty"`
	Name                   string   `json:"name,omitempty"`
	PreserveHost           bool     `json:"preserve_host,omitempty"`
	Retries                int      `json:"retries,omitempty"`
	StripURI               bool     `json:"strip_uri,omitempty"`
	UpstreamConnectTimeout int      `json:"upstream_connect_timeout,omitempty"`
	UpstreamReadTimeout    int      `json:"upstream_read_timeout,omitempty"`
	UpstreamSendTimeout    int      `json:"upstream_send_timeout,omitempty"`
	UpstreamURL            string   `json:"upstream_url,omitempty"`
}

// NewAPIConf creates new API object with sensible defaults.
func NewAPIConf() *API {
	api := API{
		Hosts:                  []string{},
		URIs:                   []string{},
		Methods:                []string{},
		HTTPIfTerminated:       true,
		HTTPSOnly:              false,
		PreserveHost:           false,
		Retries:                5,
		StripURI:               false,
		UpstreamConnectTimeout: 60000,
		UpstreamReadTimeout:    60000,
		UpstreamSendTimeout:    60000,
	}
	return &api
}

// AddHost adds a new host to the API configuration structure.
func (api *API) AddHost(host string) {
	api.Hosts = append(api.Hosts, host)
}

// GetServiceIP returns the valid IP of the microservice container.
func GetServiceIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", nil
		}
		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip.String() != "127.0.0.1" && ip.String() != "::1" {
				return ip.String(), nil
			}
		}

	}

	return "", errors.New("IP not found")
}

// getKongURL returns a full URL to the desired 'path' on the Kong Gateway.
func (kong *KongGateway) getKongURL(path string) string {
	return fmt.Sprintf("%s/%s", kong.GatewayURL, path)
}

// getUpstreamObj call the 'upstream' API on Kong and retrieves an upstream object with the given name.
// Returns the upstream object if found, or nil if there is no upstream with that name.
func (kong *KongGateway) getUpstreamObj(name string) (*upstream, error) {
	var upstreamObj upstream
	resp, err := kong.client.Get(kong.getKongURL(fmt.Sprintf("upstreams/%s", name)))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&upstreamObj); err != nil {
		return nil, err
	}
	return &upstreamObj, nil
}

// createUpstreamObj creates new upstream object on Kong.
func (kong *KongGateway) createUpstreamObj(name string, slots int) (*upstream, error) {
	var upstreamObj upstream
	form := url.Values{}

	form.Add("name", name)
	form.Add("slots", fmt.Sprintf("%d", slots))

	resp, err := kong.client.Post(kong.getKongURL("upstreams/"), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 201 {
		return nil, fmt.Errorf(resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&upstreamObj); err != nil {
		return nil, err
	}
	return &upstreamObj, nil
}

// createOrUpdateUpstream creates a new upstream object on Kong if it doesn't exist.
func (kong *KongGateway) createOrUpdateUpstream(name string, slots int) error {
	up, err := kong.getUpstreamObj(name)
	if err != nil {
		return err
	}
	if up == nil {
		if _, err = kong.createUpstreamObj(name, slots); err != nil {
			return err
		}
	}
	return nil
}

// createKongAPI creates new API object on Kong.
func (kong *KongGateway) createOrUpdateKongAPI(apiConf *API) (*API, error) {
	var result API
	form := url.Values{}

	if apiConf.Name != "" {
		form.Add("name", apiConf.Name)
	}

	if apiConf.Hosts != nil {
		form.Add("hosts", strings.Join(apiConf.Hosts, ","))
	}

	if apiConf.URIs != nil {
		form.Add("uris", strings.Join(apiConf.URIs, ","))
	}

	if apiConf.Methods != nil {
		form.Add("methods", strings.Join(apiConf.Methods, ","))
	}

	if apiConf.UpstreamURL != "" {
		form.Add("upstream_url", apiConf.UpstreamURL)
	}

	form.Add("retries", fmt.Sprintf("%d", apiConf.Retries))
	form.Add("upstream_connect_timeout", fmt.Sprintf("%d", apiConf.UpstreamConnectTimeout))
	form.Add("upstream_send_timeout", fmt.Sprintf("%d", apiConf.UpstreamSendTimeout))
	form.Add("upstream_read_timeout", fmt.Sprintf("%d", apiConf.UpstreamReadTimeout))

	form.Add("strip_uri", fmt.Sprintf("%t", apiConf.StripURI))
	form.Add("preserve_host", fmt.Sprintf("%t", apiConf.PreserveHost))
	form.Add("https_only", fmt.Sprintf("%t", apiConf.HTTPSOnly))
	form.Add("http_if_terminated", fmt.Sprintf("%t", apiConf.HTTPIfTerminated))
	var resp *http.Response
	var err error
	if apiConf.ID == "" {
		// Create API
		resp, err = kong.client.Post(kong.getKongURL("apis/"), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
	} else {
		// Update API
		req, err := http.NewRequest("PATCH", kong.getKongURL(fmt.Sprintf("apis/%s", apiConf.ID)), strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp, err = kong.client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf(resp.Status)
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	result.URIs = apiConf.URIs
	result.Methods = apiConf.Methods

	return &result, nil
}

// getAPI retrieves the API object from Kong with the given name.
// Returns the API object if found, or nil if no such object exists on Kong.
func (kong *KongGateway) getAPI(name string) (*API, error) {
	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	resp, err := kong.client.Get(kong.getKongURL(fmt.Sprintf("apis/%s", name)))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, nil
	}

	var api API
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, err
	}
	return &api, nil
}

// createOrUpdateAPI creates a new API object if it doesn't exist on Kong.
// Returns the created (or existing) object from Kong.
func (kong *KongGateway) createOrUpdateAPI(apiConf *API) (*API, error) {
	api, err := kong.getAPI(apiConf.Name)
	if err != nil {
		return nil, err
	}
	if api != nil {
		apiConf.ID = api.ID
	}
	api, err = kong.createOrUpdateKongAPI(apiConf)
	return api, err
}

// addSelfAsTarget crates a new target object on kong for this specific service with the upstream and weight.
func (kong *KongGateway) addSelfAsTarget(upstream string, port int, weight int) (*upstreamTarget, error) {
	var target upstreamTarget

	ip, err := GetServiceIP()
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Add("target", fmt.Sprintf("%s:%d", ip, port))
	form.Add("weight", fmt.Sprintf("%d", weight))

	resp, err := kong.client.Post(kong.getKongURL(fmt.Sprintf("upstreams/%s/targets", upstream)), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))

	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 201 {
		return nil, fmt.Errorf(resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return nil, err
	}

	return &target, nil
}
