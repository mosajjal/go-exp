package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

func errorHandler(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var BATCH_SIZE = 600000

func index(cmd *cobra.Command, args []string) {
	db, err := pebble.Open(cmd.Flag("path").Value.String(), &pebble.Options{})
	errorHandler(err)

	scanner := bufio.NewScanner(os.Stdin)
	c := 0
	batch := db.NewBatch()
	for scanner.Scan() {
		c++
		i := scanner.Text()
		line := strings.SplitN(i, ",", 2) //split the lines to key value by comma
		if len(line) != 2 {               // deal with single column data
			line = append(line, "")
		}

		batch.Set([]byte(line[0]), []byte(line[1]), pebble.NoSync)

		errorHandler(err)
		if c%BATCH_SIZE == 0 {
			batch.Commit(&pebble.WriteOptions{Sync: false})
			log.Printf("committing right around %s\n", i)
			batch = db.NewBatch()
		}

	}
	batch.Commit(&pebble.WriteOptions{Sync: false})
	log.Printf("finishing commits\n")
}

var QUERYRATE = 1000

func query(cmd *cobra.Command, args []string) {
	db, err := pebble.Open(cmd.Flag("path").Value.String(), &pebble.Options{})
	errorHandler(err)

	scanner := bufio.NewScanner(os.Stdin)
	c := 0
	for scanner.Scan() {
		c++
		lineInput := scanner.Text()

		fmt.Printf("querying %s\n", lineInput)
		value, closer, err := db.Get([]byte(lineInput))
		if err != nil {
			log.Printf("FAILED: %v\n", lineInput)
			continue
		}
		if err := closer.Close(); err != nil {
			log.Fatal(err)
		}
		if c%QUERYRATE == 0 {
			//todo calculate how much to sleep
			fmt.Println("SLEEPING")
			time.Sleep(time.Millisecond * 1000)
		}
		fmt.Printf("FOUND: %v: %s\n", lineInput, value)

	}
}

func remove(cmd *cobra.Command, args []string) {
	db, err := pebble.Open(cmd.Flag("path").Value.String(), &pebble.Options{})
	errorHandler(err)

	scanner := bufio.NewScanner(os.Stdin)
	c := 0
	batch := db.NewBatch()
	for scanner.Scan() {
		c++
		i := scanner.Text()
		line := strings.SplitN(i, ",", 2) //split the lines to key value by comma
		if len(line) != 2 {               // deal with single column data
			line = append(line, "")
		}

		batch.Delete([]byte(line[0]), pebble.NoSync)

		errorHandler(err)
		if c%BATCH_SIZE == 0 {
			log.Printf("deleted right around %s\n", i)
		}

	}
	log.Printf("finishing delete\n")
}

func dump(cmd *cobra.Command, args []string) {
	db, err := pebble.Open(cmd.Flag("path").Value.String(), &pebble.Options{MaxOpenFiles: 512})
	errorHandler(err)

	iter, err := db.NewIter(nil)
	errorHandler(err)
	for iter.First(); iter.Valid(); iter.Next() {
		fmt.Printf("%s,%s\n", iter.Key(), iter.Value())
	}
}
