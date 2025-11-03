package config

import (
	"net"
	"strconv"
	"strings"
)

type HostAddressParseError struct {
	message string
}

func (e HostAddressParseError) Error() string {
	return e.message
}

type HostAddress struct {
	Host string
	Port int
}

func NewHostAddress() *HostAddress {
	return &HostAddress{
		Host: "localhost",
		Port: 8080,
	}
}

func (a HostAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *HostAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	if len(hp) != 2 {
		return HostAddressParseError{message: "need address in a form host:port"}
	}
	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return HostAddressParseError{message: err.Error()}
	}

	if hp[0] != "localhost" {
		ip := net.ParseIP(hp[0])
		if ip == nil {
			return HostAddressParseError{message: "need host in ip form or localhost string"}
		}
	}
	a.Host = hp[0]
	a.Port = port
	return nil
}
