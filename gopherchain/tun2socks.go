package main

import (
	"context"
	"time"

	"go.uber.org/automaxprocs/maxprocs"

	"github.com/vishvananda/netns"
	_ "github.com/xjasonlyu/tun2socks/v2/dns"
	"github.com/xjasonlyu/tun2socks/v2/engine"
)

type Config struct {
	Mark                     int
	MTU                      int
	UDPTimeout               time.Duration
	Device                   string
	Interface                string
	LogLevel                 string
	Proxy                    string
	RestAPI                  string
	TCPSendBufferSize        string
	TCPReceiveBufferSize     string
	TCPModerateReceiveBuffer bool
	TUNPreUp                 string
	TUNPostUp                string
}

func DefaultConfig() *Config {
	return &Config{
		Mark:                     0,
		MTU:                      0,
		UDPTimeout:               0,
		Device:                   "",
		Interface:                "",
		LogLevel:                 "info",
		Proxy:                    "",
		RestAPI:                  "",
		TCPSendBufferSize:        "",
		TCPReceiveBufferSize:     "",
		TCPModerateReceiveBuffer: false,
		TUNPreUp:                 "",
		TUNPostUp:                "",
	}
}

type Option func(*Config)

func WithMark(mark int) Option {
	return func(c *Config) {
		c.Mark = mark
	}
}

func WithMTU(mtu int) Option {
	return func(c *Config) {
		c.MTU = mtu
	}
}

func WithUDPTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.UDPTimeout = timeout
	}
}

func WithDevice(device string) Option {
	return func(c *Config) {
		c.Device = device
	}
}

func WithInterface(iface string) Option {
	return func(c *Config) {
		c.Interface = iface
	}
}

func WithLogLevel(level string) Option {
	return func(c *Config) {
		c.LogLevel = level
	}
}

func WithProxy(proxy string) Option {
	return func(c *Config) {
		c.Proxy = proxy
	}
}

func WithRestAPI(api string) Option {
	return func(c *Config) {
		c.RestAPI = api
	}
}

func WithTCPSendBufferSize(size string) Option {
	return func(c *Config) {
		c.TCPSendBufferSize = size
	}
}

func WithTCPReceiveBufferSize(size string) Option {
	return func(c *Config) {
		c.TCPReceiveBufferSize = size
	}
}

func WithTCPModerateReceiveBuffer(enable bool) Option {
	return func(c *Config) {
		c.TCPModerateReceiveBuffer = enable
	}
}

func WithTUNPreUp(cmd string) Option {
	return func(c *Config) {
		c.TUNPreUp = cmd
	}
}

func WithTUNPostUp(cmd string) Option {
	return func(c *Config) {
		c.TUNPostUp = cmd
	}
}

func NewConfig(opts ...Option) *Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func NewTUN(ctx context.Context, ready chan struct{}, namespace netns.NsHandle, opts ...Option) {
	maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	// Set the network namespace
	if err := netns.Set(namespace); err != nil {
		log.Error("failed to set network namespace",
			"error", err)
	}

	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	key := &engine.Key{
		Mark:                     config.Mark,
		MTU:                      config.MTU,
		UDPTimeout:               config.UDPTimeout,
		Device:                   config.Device,
		Interface:                config.Interface,
		LogLevel:                 config.LogLevel,
		Proxy:                    config.Proxy,
		RestAPI:                  config.RestAPI,
		TCPSendBufferSize:        config.TCPSendBufferSize,
		TCPReceiveBufferSize:     config.TCPReceiveBufferSize,
		TCPModerateReceiveBuffer: config.TCPModerateReceiveBuffer,
		TUNPreUp:                 config.TUNPreUp,
		TUNPostUp:                config.TUNPostUp,
	}

	engine.Insert(key)
	engine.Start()
	defer engine.Stop()

	// send a signal to notify the caller that the TUN device is ready
	close(ready)

	<-ctx.Done()
}
