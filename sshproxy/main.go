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
	"sync"
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

type ipTime struct {
	IP net.IP
	TS time.Time
}

type configuration struct {
	Listen         string            `yaml:"listen"`
	Users          map[string]string `yaml:"users"`
	SessionPerUser int               `yaml:"session_per_user"`
	Hostkey        string            `yaml:"hostkey"`
	pKey           ssh.Signer
	// map from user to a list of IPs and timestamps.
	activeSessions map[string][]ipTime
	rwLock         sync.RWMutex
}

func (conf *configuration) connPolicy(user string, ipport net.Addr) (dropping bool) { // return true if the connection should be dropped
	ipStr, _, err := net.SplitHostPort(ipport.String())
	if err != nil {
		log.Printf("failed to parse ip:port: %s", err)
		return true
	}
	ip := net.ParseIP(ipStr)
	conf.rwLock.Lock()
	defer conf.rwLock.Unlock()
	// check if sessionperuser is configured
	if conf.SessionPerUser <= 0 {
		return false
	}

	// check if we have any sessions for this user
	// if the user has no sessions, add a new one and allow the connection
	if _, ok := conf.activeSessions[user]; !ok {
		conf.activeSessions[user] = []ipTime{{
			IP: ip,
			TS: time.Now(),
		}}
		return false
	} else {
		// if the user has sessions, check if we have a session from the same IP as before
		for i, s := range conf.activeSessions[user] {
			// if we already have a session from this IP, update the timestamp
			if s.IP.Equal(ip) {
				conf.activeSessions[user][i].TS = time.Now()
				dropping = false
			}
			// check if the session is older than 30 seconds
			if time.Since(s.TS) > 30*time.Second {
				// remove the old session
				conf.activeSessions[user] = append(conf.activeSessions[user][:i], conf.activeSessions[user][i+1:]...)
				break
			}

		}

		// check the user's current connection count against the limit
		if len(conf.activeSessions[user]) > conf.SessionPerUser {
			log.Printf("too many sessions for user %s with IPs %v", user, conf.activeSessions[user])
			return true
		}

		if !dropping {

			// double check if the IP is not already in the list
			for _, s := range conf.activeSessions[user] {
				if s.IP.Equal(ip) {
					return false
				}
			}
			// new IP for this user, add a new session
			conf.activeSessions[user] = append(conf.activeSessions[user], ipTime{
				IP: ip,
				TS: time.Now(),
			})
		}
	}

	return false
}

func main() {

	configPath := pflag.StringP("config", "c", "config.yaml", "path to the config file")
	pflag.Parse()

	// read the configuration file
	conf := configuration{}
	conf.rwLock = sync.RWMutex{}
	conf.activeSessions = make(map[string][]ipTime)
	confFile, err := os.ReadFile(*configPath)
	if err != nil {
		panic("failed to open conf.yaml: " + err.Error())
	}
	yaml.Unmarshal(confFile, &conf)
	conf.pKey, err = gossh.ParsePrivateKey([]byte(conf.Hostkey))
	if err != nil {
		panic("failed to parse hostkey: " + err.Error())
	}

	// clean up old sessions every 10 seconds
	go func() {
		for {
			time.Sleep(10 * time.Second)
			for user, _ := range conf.activeSessions {
				for i, s := range conf.activeSessions[user] {
					// check if the session is older than 30 seconds
					if time.Since(s.TS) > 30*time.Second {
						// remove the old session
						conf.activeSessions[user] = append(conf.activeSessions[user][:i], conf.activeSessions[user][i+1:]...)
						break
					}

				}
			}
		}
	}()

	// set up a SSH server with a user and password
	server := &ssh.Server{
		Addr: conf.Listen,
		Handler: func(s ssh.Session) {
			io.WriteString(s, "This server is only for port forwarding. ")
			io.WriteString(s, "please use -L, -R or -D options. Press Ctrl+C to exit.")
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
			s.Exit(0)
		},
		HostSigners: []ssh.Signer{conf.pKey},
		PasswordHandler: func(ctx ssh.Context, pass string) bool {
			if _, ok := conf.Users[ctx.User()]; ok {
				if conf.Users[ctx.User()] == pass {
					log.Printf("user %s connected from %s", ctx.User(), ctx.RemoteAddr())
					if conf.connPolicy(ctx.User(), ctx.RemoteAddr()) {
						return false
					}
					return true
				}
			}
			return false
		},
		LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
			if isLocal(dhost) || conf.connPolicy(ctx.User(), ctx.RemoteAddr()) {
				return false
			}
			return true
		},
		ReversePortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
			if isLocal(dhost) || conf.connPolicy(ctx.User(), ctx.RemoteAddr()) {
				return false
			}
			return true
		},
		IdleTimeout: 300 * time.Second,
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
