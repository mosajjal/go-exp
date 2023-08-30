// this app can not deal with the huge number of TCP_TIMEWAIT connections. in order to reuse those connections
// use net.ipv4.tcp_tw_reuse
// https://fromdual.com/huge-amount-of-time-wait-connections
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"inet.af/tcpproxy"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "sniplex",
		Short: "sniplex",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
		},
	}
	upstream := rootCmd.PersistentFlags().StringArrayP("upstream", "u", []string{}, "use it: www.google.com,142.250.66.228:443 it can also have default,127.0.0.1:4433 as default")
	bind := rootCmd.PersistentFlags().StringP("bind", "i", "0.0.0.0:443", "which IP to bind to")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
	if rootCmd.Flag("help").Changed {
		os.Exit(0)
	}

	sniDict := make(map[string]string)
	for _, v := range *upstream {
		keys := strings.Split(v, ",")
		sniDict[keys[0]] = keys[1]
	}

	var tcpProxy tcpproxy.Proxy
	for k, v := range sniDict {
		if k == "default" {
			tcpProxy.AddRoute(*bind, tcpproxy.To(v))
			continue
		}
		log.Printf("adding overide rule %s -> %s", k, v)
		tcpProxy.AddSNIRoute(*bind, k, tcpproxy.To(v))
	}
	log.Printf("starting tcpproxy on port %s", *bind)
	log.Fatal(tcpProxy.Run())
}
