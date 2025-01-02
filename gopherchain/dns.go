package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	rdns "github.com/folbricht/routedns"
	"github.com/vishvananda/netns"
)

const (
	// DNSBootStrapIPv4 is the default DNS ipv4 server to use for DoH and DoT
	DNSBootStrapIPv4 = "1.1.1.1"
	// DNSBootstrapIPv6 is the default DNS ipv6 server to use for DoH and DoT
	DNSBootstrapIPv6 = "2606:4700:4700::1111"
	// DNSTimeout is the default timeout for DNS queries
	DNSTimeout = 10 * time.Second
)

/*
NewDNSClient creates a DNS Client by parsing a URI and returning the appropriate client for it
URI string could look like below:
  - udp://1.1.1.1:53
  - udp6://[2606:4700:4700::1111]:53
  - tcp://9.9.9.9:5353
  - https://dns.adguard.com
  - quic://dns.adguard.com:8853
  - tcp-tls://dns.adguard.com:853
*/
func NewDNSClient(uri string, skipVerify bool, proxy string) (rdns.Resolver, error) {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// var dialer *rdns.Dialer
	// proxyURL, err := url.Parse(proxy)
	// if err != nil {
	// 	return nil, err
	// }
	// dialer, err = getDialerFromProxyURL(proxyURL)
	// if err != nil {
	// 	return nil, err
	// }

	switch parsedURL.Scheme {
	case "udp", "udp6":
		var host, port string
		// if port doesn't exist, use default port
		if host, port, err = net.SplitHostPort(parsedURL.Host); err != nil {
			host = parsedURL.Host
			port = "53"
		}
		Address := rdns.AddressWithDefault(host, port)

		opt := rdns.DNSClientOptions{
			UDPSize: 1300,
			// Dialer:       *dialer,
			// QueryTimeout: DNSTimeout,
		}
		id, err := rdns.NewDNSClient("id", Address, "udp", opt)
		if err != nil {
			return nil, err
		}
		return id, nil
	case "tcp", "tcp6":
		var host, port string
		// if port doesn't exist, use default port
		if host, port, err = net.SplitHostPort(parsedURL.Host); err != nil {
			host = parsedURL.Host
			port = "53"
		}

		Address := rdns.AddressWithDefault(host, port)
		opt := rdns.DNSClientOptions{
			UDPSize: 1300,
			// Dialer:    *dialer,
		}
		id, err := rdns.NewDNSClient("id", Address, "tcp", opt)
		if err != nil {
			return nil, err
		}
		return id, nil
	case "tls", "tls6", "tcp-tls", "tcp-tls6":
		tlsConfig, err := rdns.TLSClientConfig("", "", "", parsedURL.Host)
		if err != nil {
			return nil, err
		}
		opt := rdns.DoTClientOptions{
			TLSConfig:     tlsConfig,
			BootstrapAddr: DNSBootStrapIPv4,
			// LocalAddr:     ldarr,
			// Dialer:        *dialer,
		}
		id, err := rdns.NewDoTClient("id", parsedURL.Host, opt)
		if err != nil {
			return nil, err
		}
		return id, nil
	case "https":
		tlsConfig := &tls.Config{
			InsecureSkipVerify: skipVerify,
			ServerName:         strings.Split(parsedURL.Host, ":")[0],
		}

		transport := "tcp"
		opt := rdns.DoHClientOptions{
			Method:        "POST", // TODO: support anything other than POST
			TLSConfig:     tlsConfig,
			BootstrapAddr: DNSBootStrapIPv4,
			Transport:     transport,
		}
		id, err := rdns.NewDoHClient("id", parsedURL.String(), opt)
		if err != nil {
			return nil, err
		}
		return id, nil

	case "quic":
		tlsConfig := &tls.Config{
			InsecureSkipVerify: skipVerify,
			ServerName:         strings.Split(parsedURL.Host, ":")[0],
		}

		opt := rdns.DoQClientOptions{
			TLSConfig: tlsConfig,
		}
		id, err := rdns.NewDoQClient("id", parsedURL.Host, opt)
		if err != nil {
			return nil, err
		}
		return id, nil
	}
	return nil, fmt.Errorf("Can't understand the URL")
}

func ServeDNS(namespace netns.NsHandle, r *rdns.Resolver) uint {
	go func() {
		// Set the network namespace
		if err := netns.Set(namespace); err != nil {
			log.Error("failed to set network namespace",
				"error", err)
		}

		err := rdns.NewDNSListener("id", fmt.Sprintf("%s:5353", "127.0.0.1"), "udp", rdns.ListenOptions{}, *r).ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()
	return 5353
}
