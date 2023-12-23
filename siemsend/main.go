// this project has a simple function: get a stream of JSONL from stdin, and send them to Azure Sentinel
// in batches using the HTTP API Endpoint
package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "siemsend"}

// GenericOutput interface for all outputs for possible multi-output option
type GenericOutput interface {
	Send(string)
	Init() error
}

func main() {
	//todo: possibly general flags
	rootCmd.Execute()
}
