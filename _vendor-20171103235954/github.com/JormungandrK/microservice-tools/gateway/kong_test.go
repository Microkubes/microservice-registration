package gateway

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"testing"

	gock "gopkg.in/h2non/gock.v1"
)

func MatchFormParam(name string, value string, isPattern bool) gock.MatchFunc {
	return func(req *http.Request, greq *gock.Request) (bool, error) {

		receivedValue := req.PostFormValue(name)

		if &receivedValue == nil {
			return false, nil
		}
		if isPattern {
			m, err := regexp.MatchString(value, receivedValue)
			return m, err
		}
		if value != receivedValue {
			fmt.Printf("[%s] does not match [%s]\n", receivedValue, value)
		}
		return value == receivedValue, nil
	}
}

type FormMatcher struct {
	Matcher *gock.MockMatcher
}

func (fm *FormMatcher) FormParam(name string, value string) *FormMatcher {
	fm.Matcher.Add(MatchFormParam(name, value, false))
	return fm
}

func (fm *FormMatcher) FormParamPattern(name string, pattern string) *FormMatcher {
	fm.Matcher.Add(MatchFormParam(name, pattern, true))
	return fm
}

func NewFormMatcher() *FormMatcher {
	fm := FormMatcher{
		Matcher: gock.NewBasicMatcher(),
	}
	return &fm
}

func TestSelfRegisterNoUpstreamNoAPI(t *testing.T) {
	client := &http.Client{}

	defer gock.Off()

	gock.New("http://kong:8001").
		Get("/upstreams/user.api.jormugandr.org").
		Reply(404).
		JSON(map[string]string{"message": "Not Found"})

	gock.New("http://kong:8001").
		Get("/apis/user-microservice").
		Reply(404).
		JSON(map[string]string{"message": "Not Found"})

	gock.New("http://kong:8001").
		Post("/upstreams/").
		SetMatcher(NewFormMatcher().
			FormParam("name", "user.api.jormugandr.org").
			FormParam("slots", "10").
			Matcher).
		Reply(201).
		JSON(map[string]interface{}{
			"id":   "13611da7-703f-44f8-b790-fc1e7bf51b3e",
			"name": "user.api.jormugandr.org",
			"orderlist": []int{
				1,
				2,
				7,
				9,
				6,
				4,
				5,
				10,
				3,
				8,
			},
			"slots":      10,
			"created_at": 1485521710265,
		})

	gock.New("http://kong:8001").
		Post("/apis/").
		SetMatcher(NewFormMatcher().
			FormParam("name", "user-microservice").
			FormParam("hosts", "localhost,user.api.jormugandr.org").
			FormParam("upstream_url", "http://user.api.jormugandr.org:8080").
			FormParam("strip_uri", "false").
			FormParam("preserve_host", "false").
			FormParam("retries", "5").
			FormParam("upstream_connect_timeout", "60000").
			FormParam("upstream_send_timeout", "60000").
			FormParam("upstream_read_timeout", "60000").
			FormParam("https_only", "false").
			FormParam("http_if_terminated", "true").
			Matcher).
		Reply(201).
		JSON(map[string]interface{}{
			"created_at": 1488830759000,
			"hosts": []string{
				"localhost",
				"user.api.jormugandr.org",
			},
			"http_if_terminated":       true,
			"https_only":               false,
			"id":                       "6378122c-a0a1-438d-a5c6-efabae9fb969",
			"name":                     "user-microservice",
			"preserve_host":            false,
			"retries":                  5,
			"strip_uri":                false,
			"upstream_connect_timeout": 60000,
			"upstream_read_timeout":    60000,
			"upstream_send_timeout":    60000,
			"upstream_url":             "http://user.api.jormugandr.org:8080",
		})

	gock.New("http://kong:8001").
		Post("/upstreams/user.api.jormugandr.org/targets").
		SetMatcher(NewFormMatcher().
			FormParamPattern("target", "\\d+\\.\\d+\\.\\d+\\.\\d+:\\d+").
			FormParam("weight", "10").
			Matcher).
		Reply(201).
		JSON(map[string]interface{}{
			"id":          "4661f55e-95c2-4011-8fd6-c5c56df1c9db",
			"target":      "1.2.3.4:80",
			"weight":      10,
			"upstream_id": "ee3310c1-6789-40ac-9386-f79c0cb58432",
			"created_at":  1485523507446,
		})

	gock.InterceptClient(client)

	config := &MicroserviceConfig{
		MicroserviceName: "user-microservice",
		MicroservicePort: 8080,
		ServicesMaxSlots: 10,
		VirtualHost:      "user.api.jormugandr.org",
		Weight:           10,
		Hosts:            []string{"localhost", "user.api.jormugandr.org"},
	}
	gateway := NewKongGateway("http://kong:8001", client, config)

	err := gateway.SelfRegister()
	if err != nil {
		panic(err)
	}

	if gock.IsPending() {
		all := gock.GetAll()
		for _, mock := range all {
			t.Error("Mock:", mock.Request())
		}

		panic(fmt.Sprintf("Expected 5 HTTP calls to be made, but there are %d still pending", len(all)))
	}
}

func TestSelfRegisterNoAPI(t *testing.T) {
	client := &http.Client{}

	defer gock.Off()

	gock.New("http://kong:8001").
		Get("/upstreams/user.api.jormugandr.org").
		Reply(200).
		JSON(map[string]interface{}{
			"id":   "13611da7-703f-44f8-b790-fc1e7bf51b3e",
			"name": "user.api.jormugandr.org",
			"orderlist": []int{
				1,
				2,
				7,
				9,
				6,
				4,
				5,
				10,
				3,
				8,
			},
			"slots":      10,
			"created_at": 1485521710265,
		})

	gock.New("http://kong:8001").
		Get("/apis/user-microservice").
		Reply(404).
		JSON(map[string]string{"message": "Not Found"})

	gock.New("http://kong:8001").
		Post("/apis/").
		SetMatcher(NewFormMatcher().
			FormParam("name", "user-microservice").
			FormParam("hosts", "localhost,user.api.jormugandr.org").
			FormParam("upstream_url", "http://user.api.jormugandr.org:8080").
			FormParam("strip_uri", "false").
			FormParam("preserve_host", "false").
			FormParam("retries", "5").
			FormParam("upstream_connect_timeout", "60000").
			FormParam("upstream_send_timeout", "60000").
			FormParam("upstream_read_timeout", "60000").
			FormParam("https_only", "false").
			FormParam("http_if_terminated", "true").
			Matcher).
		Reply(200).
		JSON(map[string]interface{}{
			"created_at": 1488830759000,
			"hosts": []string{
				"localhost",
				"user.api.jormugandr.org",
			},
			"http_if_terminated":       true,
			"https_only":               false,
			"id":                       "6378122c-a0a1-438d-a5c6-efabae9fb969",
			"name":                     "user-microservice",
			"preserve_host":            false,
			"retries":                  5,
			"strip_uri":                false,
			"upstream_connect_timeout": 60000,
			"upstream_read_timeout":    60000,
			"upstream_send_timeout":    60000,
			"upstream_url":             "http://user.api.jormugandr.org:8080",
		})

	gock.New("http://kong:8001").
		Post("/upstreams/user.api.jormugandr.org/targets").
		SetMatcher(NewFormMatcher().
			FormParamPattern("target", "\\d+\\.\\d+\\.\\d+\\.\\d+:\\d+").
			FormParam("weight", "10").
			Matcher).
		Reply(201).
		JSON(map[string]interface{}{
			"id":          "4661f55e-95c2-4011-8fd6-c5c56df1c9db",
			"target":      "1.2.3.4:80",
			"weight":      10,
			"upstream_id": "ee3310c1-6789-40ac-9386-f79c0cb58432",
			"created_at":  1485523507446,
		})

	gock.InterceptClient(client)

	config := &MicroserviceConfig{
		MicroserviceName: "user-microservice",
		MicroservicePort: 8080,
		ServicesMaxSlots: 10,
		VirtualHost:      "user.api.jormugandr.org",
		Weight:           10,
		Hosts:            []string{"localhost", "user.api.jormugandr.org"},
	}
	gateway := NewKongGateway("http://kong:8001", client, config)

	err := gateway.SelfRegister()
	if err != nil {
		panic(err)
	}

	if gock.IsPending() {
		all := gock.GetAll()
		for _, mock := range all {
			t.Error("Mock:", mock.Request())
		}

		panic(fmt.Sprintf("Expected 4 HTTP calls to be made, but there are %d still pending", len(all)))
	}
}

func TestSelfRegisterNoUpstream(t *testing.T) {
	client := &http.Client{}

	defer gock.Off()

	gock.New("http://kong:8001").
		Get("/upstreams/user.api.jormugandr.org").
		Reply(404).
		JSON(map[string]string{"message": "Not Found"})

	gock.New("http://kong:8001").
		Get("/apis/user-microservice").
		Reply(200).
		JSON(map[string]interface{}{
			"created_at": 1488830759000,
			"hosts": []string{
				"localhost",
				"user.api.jormugandr.org",
			},
			"http_if_terminated":       true,
			"https_only":               false,
			"id":                       "6378122c-a0a1-438d-a5c6-efabae9fb969",
			"name":                     "user-microservice",
			"preserve_host":            false,
			"retries":                  5,
			"strip_uri":                false,
			"upstream_connect_timeout": 60000,
			"upstream_read_timeout":    60000,
			"upstream_send_timeout":    60000,
			"upstream_url":             "http://user.api.jormugandr.org:8080",
		})
	gock.New("http://kong:8001").
		Patch("/apis/6378122c-a0a1-438d-a5c6-efabae9fb969").
		Reply(200).
		JSON(map[string]interface{}{
			"created_at": 1488830759000,
			"hosts": []string{
				"localhost",
				"user.api.jormugandr.org",
			},
			"http_if_terminated":       true,
			"https_only":               false,
			"id":                       "6378122c-a0a1-438d-a5c6-efabae9fb969",
			"name":                     "user-microservice",
			"preserve_host":            false,
			"retries":                  5,
			"strip_uri":                false,
			"upstream_connect_timeout": 60000,
			"upstream_read_timeout":    60000,
			"upstream_send_timeout":    60000,
			"upstream_url":             "http://user.api.jormugandr.org:8080",
		})

	gock.New("http://kong:8001").
		Post("/upstreams/").
		SetMatcher(NewFormMatcher().
			FormParam("name", "user.api.jormugandr.org").
			FormParam("slots", "10").
			Matcher).
		Reply(201).
		JSON(map[string]interface{}{
			"id":   "13611da7-703f-44f8-b790-fc1e7bf51b3e",
			"name": "user.api.jormugandr.org",
			"orderlist": []int{
				1,
				2,
				7,
				9,
				6,
				4,
				5,
				10,
				3,
				8,
			},
			"slots":      10,
			"created_at": 1485521710265,
		})

	gock.New("http://kong:8001").
		Post("/upstreams/user.api.jormugandr.org/targets").
		SetMatcher(NewFormMatcher().
			FormParamPattern("target", "\\d+\\.\\d+\\.\\d+\\.\\d+:\\d+").
			FormParam("weight", "10").
			Matcher).
		Reply(201).
		JSON(map[string]interface{}{
			"id":          "4661f55e-95c2-4011-8fd6-c5c56df1c9db",
			"target":      "1.2.3.4:80",
			"weight":      10,
			"upstream_id": "ee3310c1-6789-40ac-9386-f79c0cb58432",
			"created_at":  1485523507446,
		})

	gock.InterceptClient(client)

	config := &MicroserviceConfig{
		MicroserviceName: "user-microservice",
		MicroservicePort: 8080,
		ServicesMaxSlots: 10,
		VirtualHost:      "user.api.jormugandr.org",
		Weight:           10,
		Hosts:            []string{"localhost", "user.api.jormugandr.org"},
	}
	gateway := NewKongGateway("http://kong:8001", client, config)

	err := gateway.SelfRegister()
	if err != nil {
		panic(err)
	}

	if gock.IsPending() {
		all := gock.GetAll()
		for _, mock := range all {
			t.Error("Mock:", mock.Request())
		}

		panic(fmt.Sprintf("Expected 4 HTTP calls to be made, but there are %d still pending", len(all)))
	}
}

func TestSelfRegisterWithAPIAndUpstream(t *testing.T) {
	client := &http.Client{}

	defer gock.Off()

	gock.New("http://kong:8001").
		Get("/upstreams/user.api.jormugandr.org").
		Reply(200).
		JSON(map[string]interface{}{
			"id":   "13611da7-703f-44f8-b790-fc1e7bf51b3e",
			"name": "user.api.jormugandr.org",
			"orderlist": []int{
				1,
				2,
				7,
				9,
				6,
				4,
				5,
				10,
				3,
				8,
			},
			"slots":      10,
			"created_at": 1485521710265,
		})

	gock.New("http://kong:8001").
		Get("/apis/user-microservice").
		Reply(200).
		JSON(map[string]interface{}{
			"created_at": 1488830759000,
			"hosts": []string{
				"localhost",
				"user.api.jormugandr.org",
			},
			"http_if_terminated":       true,
			"https_only":               false,
			"id":                       "6378122c-a0a1-438d-a5c6-efabae9fb969",
			"name":                     "user-microservice",
			"preserve_host":            false,
			"retries":                  5,
			"strip_uri":                false,
			"upstream_connect_timeout": 60000,
			"upstream_read_timeout":    60000,
			"upstream_send_timeout":    60000,
			"upstream_url":             "http://user.api.jormugandr.org:8080",
		})
	gock.New("http://kong:8001").
		Patch("/apis/6378122c-a0a1-438d-a5c6-efabae9fb969").
		Reply(200).
		JSON(map[string]interface{}{
			"created_at": 1488830759000,
			"hosts": []string{
				"localhost",
				"user.api.jormugandr.org",
			},
			"http_if_terminated":       true,
			"https_only":               false,
			"id":                       "6378122c-a0a1-438d-a5c6-efabae9fb969",
			"name":                     "user-microservice",
			"preserve_host":            false,
			"retries":                  5,
			"strip_uri":                false,
			"upstream_connect_timeout": 60000,
			"upstream_read_timeout":    60000,
			"upstream_send_timeout":    60000,
			"upstream_url":             "http://user.api.jormugandr.org:8080",
		})

	gock.New("http://kong:8001").
		Post("/upstreams/user.api.jormugandr.org/targets").
		SetMatcher(NewFormMatcher().
			FormParamPattern("target", "\\d+\\.\\d+\\.\\d+\\.\\d+:\\d+").
			FormParam("weight", "10").
			Matcher).
		Reply(201).
		JSON(map[string]interface{}{
			"id":          "4661f55e-95c2-4011-8fd6-c5c56df1c9db",
			"target":      "1.2.3.4:80",
			"weight":      10,
			"upstream_id": "ee3310c1-6789-40ac-9386-f79c0cb58432",
			"created_at":  1485523507446,
		})

	gock.InterceptClient(client)

	config := &MicroserviceConfig{
		MicroserviceName: "user-microservice",
		MicroservicePort: 8080,
		ServicesMaxSlots: 10,
		VirtualHost:      "user.api.jormugandr.org",
		Weight:           10,
		Hosts:            []string{"localhost", "user.api.jormugandr.org"},
	}
	gateway := NewKongGateway("http://kong:8001", client, config)

	err := gateway.SelfRegister()
	if err != nil {
		panic(err)
	}

	if gock.IsPending() {
		all := gock.GetAll()
		for _, mock := range all {
			t.Error("Mock:", mock.Request())
		}

		panic(fmt.Sprintf("Expected 3 HTTP calls to be made, but there are %d still pending", len(all)))
	}
}

func TestNewKongGateway(t *testing.T) {
	config := &MicroserviceConfig{
		MicroserviceName: "user-microservice",
		MicroservicePort: 8080,
		ServicesMaxSlots: 10,
		VirtualHost:      "user.api.jormugandr.org",
		Weight:           10,
		Hosts:            []string{"localhost", "user.api.jormugandr.org"},
	}
	client := &http.Client{}
	gateway := NewKongGateway("http://kong:8001", client, config)
	if gateway == nil {
		panic("Gateway must be created")
	}
	if gateway.GatewayURL != "http://kong:8001" {
		panic("Wrong API Gateway URL")
	}
}

func TestNewKongGatewayFromConfigFile(t *testing.T) {
	content := "{\"name\": \"user-microservice\",\"port\": 8080,\"virtual_host\": \"user.service.jormugandr.org\",\"hosts\": [\"localhost\", \"user.service.jormugandr.org\"],\"slots\": 100,\"weight\": 15}"
	tmpf, err := ioutil.TempFile("", "service-config.json")
	if err != nil {
		panic("Cannot create temporary service config file!")
	}
	defer os.Remove(tmpf.Name())

	tmpf.WriteString(content)
	tmpf.Sync()

	gateway, err := NewKongGatewayFromConfigFile("http://kong:8080", &http.Client{}, tmpf.Name())
	if err != nil {
		panic(err)
	}
	if gateway == nil {
		panic("Gateway must be created")
	}
	if gateway.GatewayURL != "http://kong:8080" {
		panic("Wrong API Gateway URL")
	}
}

func TestGetServiceIP(t *testing.T) {
	ip, err := GetServiceIP()
	if err != nil {
		panic(err)
	}

	if ip == "" {
		panic("IP not found")
	}

	t.Logf("Service IP: %s", ip)
}

func TestUnregister(t *testing.T) {
	client := &http.Client{}

	defer gock.Off()

	gock.New("http://kong:8001").
		Post("/upstreams/user.api.jormugandr.org/targets").
		SetMatcher(NewFormMatcher().
			FormParamPattern("target", "\\d+\\.\\d+\\.\\d+\\.\\d+:\\d+").
			FormParam("weight", "0").
			Matcher).
		Reply(201).
		JSON(map[string]interface{}{
			"id":          "4661f55e-95c2-4011-8fd6-c5c56df1c9db",
			"target":      "1.2.3.4:80",
			"weight":      10,
			"upstream_id": "ee3310c1-6789-40ac-9386-f79c0cb58432",
			"created_at":  1485523507446,
		})

	gock.InterceptClient(client)

	config := &MicroserviceConfig{
		MicroserviceName: "user-microservice",
		MicroservicePort: 8080,
		ServicesMaxSlots: 10,
		VirtualHost:      "user.api.jormugandr.org",
		Weight:           10,
		Hosts:            []string{"localhost", "user.api.jormugandr.org"},
	}
	gateway := NewKongGateway("http://kong:8001", client, config)

	err := gateway.Unregister()
	if err != nil {
		panic(err)
	}

	if gock.IsPending() {
		all := gock.GetAll()
		for _, mock := range all {
			t.Error("Mock:", mock.Request())
		}

		panic(fmt.Sprintf("Expected an HTTP call to upstream/{}/targets"))
	}
}
