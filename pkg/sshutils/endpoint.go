package sshutils

import (
	"fmt"

	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

type EndpointIface interface {
	String() string
	GetSSHConfig() (*ssh.ClientConfig, error)
}

type Endpoint struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
	PublicKey  string
}

func NewEndpoint(s string) *Endpoint {
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

	return &endpoint
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

func (e *Endpoint) GetSSHConfig() (*ssh.ClientConfig, error) {
	key, err := ssh.ParsePrivateKey([]byte(e.PrivateKey))
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: e.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}
