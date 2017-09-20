// Code generated by goagen v1.3.0, DO NOT EDIT.
//
// API "user": user Resource Client
//
// Command:
// $ goagen
// --design=github.com/JormungandrK/microservice-registration/design
// --out=$(GOPATH)/src/github.com/JormungandrK/microservice-registration
// --version=v1.2.0-dirty

package client

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// RegisterUserPath computes a request path to the register action of user.
func RegisterUserPath() string {

	return fmt.Sprintf("/users/register")
}

// Creates user
func (c *Client) RegisterUser(ctx context.Context, path string, payload *UserPayload, contentType string) (*http.Response, error) {
	req, err := c.NewRegisterUserRequest(ctx, path, payload, contentType)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(ctx, req)
}

// NewRegisterUserRequest create the request corresponding to the register action endpoint of the user resource.
func (c *Client) NewRegisterUserRequest(ctx context.Context, path string, payload *UserPayload, contentType string) (*http.Request, error) {
	var body bytes.Buffer
	if contentType == "" {
		contentType = "*/*" // Use default encoder
	}
	err := c.Encoder.Encode(payload, &body, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to encode body: %s", err)
	}
	scheme := c.Scheme
	if scheme == "" {
		scheme = "http"
	}
	u := url.URL{Host: c.Host, Scheme: scheme, Path: path}
	req, err := http.NewRequest("POST", u.String(), &body)
	if err != nil {
		return nil, err
	}
	header := req.Header
	if contentType == "*/*" {
		header.Set("Content-Type", "application/json")
	} else {
		header.Set("Content-Type", contentType)
	}
	return req, nil
}
