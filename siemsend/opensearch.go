package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Opensearch struct {
	Endpoint  string
	Index     string
	BatchSize uint
	Compress  bool
}

var _ = Opensearch{}.init()

func (s Opensearch) init() error {
	OpensearchCmd := &cobra.Command{
		Use:   "opensearch [arguments]",
		Short: "send input data to Opensearch",
		Long:  `make sure your data is in jsonl format, meaning each line is a separate json object.`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			s.Send(cmd, args)
		},
	}
	flags := OpensearchCmd.Flags()
	flags.StringVarP(&s.Endpoint, "endpoint", "", "", "endpoint")
	flags.StringVarP(&s.Index, "index", "", "", "index")
	flags.UintVarP(&s.BatchSize, "batch_size", "", 100, "batch size")
	flags.BoolVarP(&s.Compress, "compress", "", false, "compress")

	rootCmd.AddCommand(OpensearchCmd)
	return nil
}

func (s Opensearch) Send(cmd *cobra.Command, args []string) {
	// tr := &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }
	// httpClient := &http.Client{Transport: tr}

	client, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{s.Endpoint},
		Username:  "admin", // For testing only. Don't store credentials in code.
		Password:  "admin",
	})

	if err != nil {
		log.Fatal(err)
	}

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	ctx := context.Background()

	// Use the IndexExists service to check if a specified index exists.
	settings := strings.NewReader(`{
		'settings': {
			'index': {
				'number_of_shards': 1,
				'number_of_replicas': 0
				}
			}
		}`)

	res, err := opensearchapi.IndicesCreateRequest{
		Index: s.Index,
		Body:  settings,
	}.Do(ctx, client)
	if err != nil {
		log.Warn(err)
	}
	if res.StatusCode != 201 {
		log.Warnf("Index creation failed")
	}
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 640*1024)
	scanner.Buffer(buf, 1024*1024*10)

	cnt := 0
	for scanner.Scan() {
		cnt++
		res, err := opensearchapi.IndexRequest{
			Index:      s.Index,
			Body:       bytes.NewReader([]byte(scanner.Text())),
			Refresh:    "false",
			DocumentID: fmt.Sprintf("%d", cnt),
			Timeout:    2 * time.Second,
		}.Do(ctx, client)
		if err != nil {
			log.Warn(err)
		}
		if res.StatusCode != 201 {
			log.Warnf("insert failed")
		}
		// _, err := client.Index().
		// 	Index(s.Index).
		// 	Type("_doc").
		// 	BodyString(scanner.Text()).
		// 	Do(ctx)

		if cnt%int(s.BatchSize) == 0 {
			// client.Flush().Index(s.Index).Do(ctx)
			log.Infoln(cnt)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}
