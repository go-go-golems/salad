package saleae

import (
	"net"
	"strconv"
	"time"
)

type Config struct {
	Host    string
	Port    int
	Timeout time.Duration
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}
