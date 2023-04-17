package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// sample YAML config
// webserver:
//   - listen: "0.0.0.0:3000"
//   - enable_tls: false
//   - tls_cert: "/path/to/cert.pem"
//   - tls_key: "/path/to/key.pem"
//   - tls_client_auth: false
//   - tls_client_ca: "/path/to/ca.pem"
//   - tls_client_ca_path: "/path/to/ca_dir"
//   - auth_provider: basic # options: basic, azuread
// 	 - users: # used only if auth_provider is basic
//     - "admin": "admin"
//     - "user": "user"
//   - azuread_key: "AZUREAD_KEY" # used only if auth_provider is azuread
//   - azuread_secret: "AZUREAD_SECRET" # used only if auth_provider is azuread
// 	 - azuread_callback: "http://localhost:3000/auth/azuread/callback" # used only if auth_provider is azuread

// service:
//   - timeout_default: 300
//   - timeout_max: 3000
//   - timeout_min: 60
//   - provider: "docker" # only option for now
//   - docker_image: "docker.io/accetto/ubuntu-vnc-xfce-chromium-g3:latest"
//   - docker_port: "6901"

type Config struct {
	Webserver struct {
		Listen          string `yaml:"listen"`
		EnableTLS       bool   `yaml:"enable_tls"`
		TLSCert         string `yaml:"tls_cert"`
		TLSKey          string `yaml:"tls_key"`
		TLSClientAuth   bool   `yaml:"tls_client_auth"`
		TLSClientCA     string `yaml:"tls_client_ca"`
		TLSClientCAPath string `yaml:"tls_client_ca_path"`
		AuthProvider    string `yaml:"auth_provider"`
		Users           map[string]string
		AzureADKey      string `yaml:"azuread_key"`
		AzureADSecret   string `yaml:"azuread_secret"`
		AzureADCallback string `yaml:"azuread_callback"`
	} `yaml:"webserver"`
	Service struct {
		TimeoutDefault int    `yaml:"timeout_default"`
		TimeoutMax     int    `yaml:"timeout_max"`
		TimeoutMin     int    `yaml:"timeout_min"`
		Provider       string `yaml:"provider"`
		DockerImage    string `yaml:"docker_image"`
		DockerPort     string `yaml:"docker_port"`
	} `yaml:"service"`
}

func readConfig(pathFile string) *Config {
	var config Config

	// open config file
	file, err := os.Open(pathFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// read config file
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}
	return &config
}
