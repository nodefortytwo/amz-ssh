package sshutils

import (
	"fmt"

	"strconv"
	"strings"
)

type Endpoint struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
}

func NewEndpoint(s string) Endpoint {
	endpoint := Endpoint{
		Host: s,
	}
	if parts := strings.Split(endpoint.Host, "@"); len(parts) > 1 {
		endpoint.User = parts[0]
		endpoint.Host = parts[1]
	}
	if parts := strings.Split(endpoint.Host, ":"); len(parts) > 1 {
		endpoint.Host = parts[0]
		endpoint.Port, _ = strconv.Atoi(parts[1])
	}

	if endpoint.Port == 0 {
		endpoint.Port = 22
	}

	return endpoint
}

func (e Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}
