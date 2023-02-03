// sshproxy is a SSH server that only allows port forwarding.
// it reads the configuration from a yaml file and can limit the number of sessions per user.
// configuration file example:
/*
	listen: ":2222"
	users:
	  user1: password1
	  user2: password2
	session_per_user: 2
	hostkey: |
	  -----BEGIN OPENSSH PRIVATE KEY-----
	  ...
	  -----END OPENSSH PRIVATE KEY-----

some tips to get you started:

	-  to generate a hostkey use: `ssh-keygen -t ed25519 -f hostkey`
	-  to run the server use: `go run main.go -c config.yaml`
	-  to connect to the server use `ssh -D 1080 -p 2222 user1@localhost`
	-  to generate a secure password use: `openssl rand -base64 24`
*/
package main

import (
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/spf13/pflag"
	gossh "golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

func isLocal(host string) bool {
	hostIP := net.ParseIP(host)
	if hostIP.IsLoopback() || hostIP.IsLinkLocalUnicast() || hostIP.IsLinkLocalMulticast() || hostIP.IsPrivate() {
		return true
	}
	return false
}

type configuration struct {
	Listen         string            `yaml:"listen"`
	Users          map[string]string `yaml:"users"`
	SessionPerUser int               `yaml:"session_per_user"`
	Hostkey        string            `yaml:"hostkey"`
	pKey           ssh.Signer
	activeSessions map[string]int
}

func main() {

	configPath := pflag.StringP("config", "c", "config.yaml", "path to the config file")
	pflag.Parse()

	// read the configuration file
	conf := configuration{}
	conf.activeSessions = make(map[string]int)
	confFile, err := os.ReadFile(*configPath)
	if err != nil {
		panic("failed to open conf.yaml: " + err.Error())
	}
	yaml.Unmarshal(confFile, &conf)
	conf.pKey, err = gossh.ParsePrivateKey([]byte(conf.Hostkey))
	if err != nil {
		panic("failed to parse hostkey: " + err.Error())
	}

	// set up a SSH server with a user and password
	server := &ssh.Server{
		Addr: conf.Listen,
		Handler: func(s ssh.Session) {
			t1 := time.Now()
			// check if the user has reached the maximum number of sessions
			if _, ok := conf.activeSessions[s.User()]; ok {
				conf.activeSessions[s.User()]++
			} else {
				conf.activeSessions[s.User()] = 1
			}
			if conf.activeSessions[s.User()] > conf.SessionPerUser {
				io.WriteString(s, "You have reached the maximum number of sessions. Please wait for one of your sessions to end.")
				s.Exit(1)
			}

			io.WriteString(s, "This server is only for port forwarding. please use -L, -R or -D options with ssh. Press Ctrl+C to exit.")
			log.Println("new session from " + s.User())
			buf := make([]byte, 1)
			for {
				_, err := s.Read(buf)
				if err != nil {
					break
				}
				// wait for the user to press Ctrl+C
				if buf[0] == 3 {
					break
				}
			}
			// remove the session from the activeSessions map
			conf.activeSessions[s.User()]--
			log.Println("session from " + s.User() + " ended. duration: " + time.Since(t1).String())
			s.Exit(0)
		},
		PasswordHandler: func(ctx ssh.Context, pass string) bool {

			if _, ok := conf.Users[ctx.User()]; ok {
				// calculate the sha256 of the password
				if conf.Users[ctx.User()] == pass {
					return true
				}
			}
			return false
		},
		HostSigners: []ssh.Signer{conf.pKey},
		LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
			if isLocal(dhost) {
				return false
			}
			return true
		},
		ReversePortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
			if isLocal(dhost) {
				return false
			}
			return true
		},

		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip": ssh.DirectTCPIPHandler,
			"session":      ssh.DefaultSessionHandler,
		},
	}

	// start the SSH server
	log.Println("starting SSH server on " + conf.Listen + ". Press Ctrl+C to exit.")
	if err := server.ListenAndServe(); err != nil {
		panic("failed to start server: " + err.Error())
	}
}
