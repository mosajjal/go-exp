package main

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Webserver struct {
		Listen          string            `yaml:"listen"`
		EnableTLS       bool              `yaml:"enable_tls"`
		TLSCert         string            `yaml:"tls_cert"`
		TLSKey          string            `yaml:"tls_key"`
		TLSClientAuth   bool              `yaml:"tls_client_auth"`
		TLSClientCA     string            `yaml:"tls_client_ca"`
		TLSClientCAPath string            `yaml:"tls_client_ca_path"`
		AuthProvider    string            `yaml:"auth_provider"`
		Users           map[string]string `yaml:"users"`
		AzureADKey      string            `yaml:"azuread_key"`
		AzureADSecret   string            `yaml:"azuread_secret"`
		AzureADCallback string            `yaml:"azuread_callback"`
		TimeoutDefault  time.Duration     `yaml:"timeout_default"`
		TimeoutMax      time.Duration     `yaml:"timeout_max"`
	} `yaml:"webserver"`
	Services map[string]struct {
		Provider    string `yaml:"provider"`
		DockerImage string `yaml:"docker_image"`
		DockerPort  string `yaml:"docker_port"`
	} `yaml:"services"`
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
