package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/pkg/profile"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "sctptunnel",
		Short: "Tunnel TCP over SCTP",
		Long:  `it's like socat but over SCTP.`,
	}

	startProfile := rootCmd.PersistentFlags().Bool("profile", false, "enable profiling")

	client := &cobra.Command{
		Use:   "client",
		Short: "Run as client",
		Long:  `Run as client`,
		Run: func(cmd *cobra.Command, args []string) {
			go runClient(cmd)
		},
	}
	// add flags
	client.Flags().StringP("server", "s", "0.0.0.0:1080", "TCP listen address")
	client.Flags().StringP("localaddr", "l", "", "Enforce source IP. will use the os default if empty")
	client.Flags().StringP("upstream", "u", "127.0.0.1:4444", "destination SCTP address")

	// add client to root command
	rootCmd.AddCommand(client)

	server := &cobra.Command{
		Use:   "server",
		Short: "Run as server",
		Long:  `Run as server`,
		Run: func(cmd *cobra.Command, args []string) {
			go runServer(cmd)
		},
	}
	// add flags
	server.Flags().StringP("server", "s", "0.0.0.0:4444", "SCTP listen address")
	server.Flags().StringP("upstream", "u", "127.0.0.1:1080", "destination TCP address")

	// add server to root command
	rootCmd.AddCommand(server)

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *startProfile {
		// start profiling
		defer profile.Start().Stop()
	}

	// handle signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	os.Exit(0)

}
