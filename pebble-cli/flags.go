package main

import (
	"github.com/spf13/cobra"
)

func main() {
	var databasePath string

	var cmdIndex = &cobra.Command{
		Use:   "index [arguments]",
		Short: "Index the csv from stdin to the database",
		Long: `For any csv file being inserted into the database, the first column will be used as key,
		and the rest will be used as value. the delimeter is always ','`,
		Args: cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			index(cmd, args)
		},
	}

	var cmdRemove = &cobra.Command{
		Use:   "remove [arguments]",
		Short: "remove list of keys coming from stdin",
		Long: `the input can be a csv or a list of strings, one in each line.
		if the input is csv, only the first column will be considered as key`,
		Args: cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			remove(cmd, args)
		},
	}

	var cmdQuery = &cobra.Command{
		Use:   "query [arguments]",
		Short: "query a list of keys coming from stdin against db",
		Long:  `Queries the db against the provided key(s)`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			query(cmd, args)
		},
	}

	var cmdDump = &cobra.Command{
		Use:   "dump [arguments]",
		Short: "dump all the keys and values in the db",
		Long:  `dump the database as a csv document`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			dump(cmd, args)
		},
	}

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(cmdIndex, cmdRemove, cmdQuery, cmdDump)
	rootCmd.PersistentFlags().StringVarP(&databasePath, "path", "p", "./mydb", "database folder")
	rootCmd.MarkPersistentFlagRequired("path")
	rootCmd.Execute()
}
