package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
)

type splunkConfig struct {
	Endpoint      []string
	SkipTLSVerify bool
	Token         string
	Index         string
	Proxy         string
	Source        string
	SourceType    string
	connections   map[string]splunkConnection
}

type splunkConnection struct {
	Client    *splunk.Client
	Unhealthy uint
	Err       error
}

func (c splunkConfig) Init() error {
	splunkCmd := &cobra.Command{
		Use:   "splunk [arguments]",
		Short: "send input data to SplunkHEC",
		Long:  `make sure your data is in jsonl format, meaning each line is a separate json object. They'll be reconstructed into a JSON array and sent to Microsoft Sentinel.`,
		Args:  cobra.ExactArgs(0),
		Run: func(_ *cobra.Command, _ []string) {
			c.Output()
		},
	}
	flags := splunkCmd.Flags()
	flags.StringSliceVarP(&c.Endpoint, "endpoint", "e", []string{}, "Splunk HEC endpoint")
	flags.BoolVarP(&c.SkipTLSVerify, "skip-tls-verify", "k", false, "Skip TLS verification")
	flags.StringVarP(&c.Token, "token", "t", "", "Splunk HEC token")
	flags.StringVarP(&c.Index, "index", "i", "", "Splunk index")
	flags.StringVarP(&c.Proxy, "proxy", "p", "", "Proxy URL")
	flags.StringVarP(&c.Source, "source", "s", "", "Splunk source")
	flags.StringVarP(&c.SourceType, "sourcetype", "y", "", "Splunk sourcetype")
	c.connections = make(map[string]splunkConnection)
	rootCmd.AddCommand(splunkCmd)
	return nil
}

func (c splunkConfig) connectMultiSplunkRetry() {
	for _, splunkEndpoint := range c.Endpoint {
		go c.connectSplunkRetry(splunkEndpoint)
	}
}

func (c splunkConfig) connectSplunkRetry(splunkEndpoint string) {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	defer tick.Stop()
	for range tick.C {
		// check to see if the connection exists
		if conn, ok := c.connections[splunkEndpoint]; ok {
			if conn.Unhealthy != 0 {
				log.Warnf("Connection is unhealthy")
				c.connections[splunkEndpoint] = c.connectSplunk(splunkEndpoint)
			}
		} else {
			log.Warnf("new splunk endpoint %s", splunkEndpoint)
			c.connections[splunkEndpoint] = c.connectSplunk(splunkEndpoint)
		}
	}
}

func (c splunkConfig) connectSplunk(splunkEndpoint string) splunkConnection {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: c.SkipTLSVerify}}
	httpClient := &http.Client{Timeout: time.Second * 20, Transport: tr}

	if c.Proxy != "" {
		proxyURL, err := url.Parse(c.Proxy)
		if err != nil {
			panic(err)
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	splunkURL := splunkEndpoint
	if !strings.HasSuffix(splunkEndpoint, "/services/collector") {
		splunkURL = fmt.Sprintf("%s/services/collector", splunkEndpoint)
	}

	// we won't define sourcetype and index here, because we want to be able to do that per write
	client := splunk.NewClient(
		httpClient,
		splunkURL,
		c.Token,
		c.Source,
		c.SourceType,
		c.Index,
	)
	err := client.CheckHealth()
	unhealthy := uint(0)
	if err != nil {
		unhealthy++
	}
	myConn := splunkConnection{Client: client, Unhealthy: unhealthy, Err: err}
	log.Warnf("new splunk connection")
	return myConn
}

func (c splunkConfig) Output() {

	log.Infof("Connecting to Splunk endpoints")
	c.connectMultiSplunkRetry()

	rand.Seed(time.Now().Unix())

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if scanner.Scan() {
		}
	}

}

var _ = splunkConfig{}.Init()
