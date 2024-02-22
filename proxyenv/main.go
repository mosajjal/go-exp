package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/armon/go-radix"
	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Listen         string   `env:"LISTEN"`
	BasicAuth      string   `env:"AUTH"`
	AllowedDomains []string `env:"ALLOWED, delimiter=;"` //these are domain suffixes only
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func main() {
	// a http proxy that is fully configured by environment variables
	var config Config
	envconfig.Process(context.Background(), &config)

	tree := radix.New()
	for _, word := range config.AllowedDomains {
		// reverse the domain name to make it a suffix
		tree.Insert(Reverse(word), word)
	}

	log.Printf("Starting proxy on %s", config.Listen)
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	if config.BasicAuth != "" {
		userPass := strings.Split(config.BasicAuth, ":")
		if len(userPass) != 2 {
			log.Fatal("Invalid AUTH format. Use 'user:password'")
		}
		proxy.OnRequest().Do(auth.Basic("my_realm", func(user, passwd string) bool {
			return user == userPass[0] && passwd == userPass[1]
		}))
		proxy.OnRequest().HandleConnect(auth.BasicConnect("my_realm", func(user, passwd string) bool {
			return user == userPass[0] && passwd == userPass[1]
		}))
	}

	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			if _, _, ok := tree.LongestPrefix(Reverse(r.Host)); !ok {
				return r, goproxy.NewResponse(r, goproxy.ContentTypeText, http.StatusForbidden, "Forbidden")
			}
			return r, nil
		},
	)
	proxy.OnRequest().HandleConnectFunc(
		func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
			h := strings.SplitN(host, ":", 2)[0]
			if _, _, ok := tree.LongestPrefix(Reverse(h)); !ok {
				return goproxy.RejectConnect, host
			}
			return goproxy.OkConnect, host
		},
	)

	log.Fatal(http.ListenAndServe(config.Listen, proxy))
}
