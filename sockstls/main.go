package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/armon/go-socks5"
	"golang.org/x/net/proxy"
)

func main() {

	cert := flag.String("cert", "", "cert file for tls. empty means tcp")
	key := flag.String("key", "", "key file for tls. empty means tcp")
	listen := flag.String("listen", "", "listen address. example: 0.0.0.0:1083")
	upstream := flag.String("upstream", "", "optional upstream socks5 uri")
	auth := flag.String("auth", "", "user1:pass1,user2:pass2")
	flag.Parse()

	var d proxy.Dialer
	if *upstream != "" {
		var err error
		u, err := url.Parse(*upstream)
		if err != nil {
			log.Println(err)
			return
		}
		d, err = proxy.FromURL(u, proxy.Direct)
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		d = proxy.Direct
	}

	// TODO: support unauthenticated mode
	var authSocks socks5.StaticCredentials
	if *auth != "" {
		authSocks = make(map[string]string)
		for _, v := range strings.Split(*auth, ",") {
			pair := strings.Split(v, ":")
			if len(pair) != 2 {
				log.Println("Invalid auth format")
				return
			}
			authSocks[pair[0]] = pair[1]
		}
	}

	var listener net.Listener
	if *cert == "" || *key == "" {
		// Plain listener
		var err error
		listener, err = net.Listen("tcp", *listen)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Listening on tcp: ", *listen)
	} else {
		cer, err := tls.LoadX509KeyPair(*cert, *key)
		if err != nil {
			log.Println(err)
			return
		}
		// TLS listener
		listener, err = tls.Listen("tcp", *listen, &tls.Config{
			Certificates: []tls.Certificate{cer},
		})
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Listening on tls: ", *listen)
	}

	// Create a SOCKS5 server
	conf := &socks5.Config{Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
		log.Printf("[INFO] dialing %s", addr)
		return d.Dial(network, addr)
	}, Logger: log.New(log.Writer(), "", log.LstdFlags),
		AuthMethods: []socks5.Authenticator{socks5.UserPassAuthenticator{
			Credentials: authSocks,
		}},
	}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}
	log.Println("starting SOCKS5 server")

	// Create SOCKS5 proxy on localhost port 8000
	if err := server.Serve(listener); err != nil {
		panic(err)
	}

}
