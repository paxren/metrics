package config

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

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
		return errors.New("need address in a form host:port")
	}
	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return err
	}

	if hp[0] != "localhost" {
		ip := net.ParseIP(hp[0])
		if ip == nil {
			return errors.New("need host in ip form or localhost string")
		}
	}
	a.Host = hp[0]
	a.Port = port
	return nil
}
