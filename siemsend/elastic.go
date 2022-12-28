package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type elastic struct {
	Endpoint string
	Index    string
	Compress bool
}

var _ = elastic{}.init()

func (s elastic) init() error {
	ElasticCmd := &cobra.Command{
		Use:   "elastic [arguments]",
		Short: "send input data to Elasticsearch",
		Long:  `make sure your data is in jsonl format, meaning each line is a separate json object.`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			s.Send(cmd, args)
		},
	}
	flags := ElasticCmd.Flags()
	flags.StringVarP(&s.Endpoint, "endpoint", "", "", "endpoint")
	flags.StringVarP(&s.Index, "index", "", "", "index")
	flags.BoolVarP(&s.Compress, "compress", "", false, "compress")

	rootCmd.AddCommand(ElasticCmd)
	return nil
}

func (s elastic) Send(cmd *cobra.Command, args []string) {
	// tr := &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }
	// httpClient := &http.Client{Transport: tr}

	cfg := elasticsearch.Config{
		Addresses: []string{
			s.Endpoint,
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}, CompressRequestBody: s.Compress,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	ctx := context.Background()

	// Use the IndexExists service to check if a specified index exists.
	exists, err := esapi.IndicesExistsRequest{
		Index: []string{s.Index},
	}.Do(ctx, client)

	if err != nil {
		log.Fatal(err)
	}

	if exists.IsError() {
		// Create a new index.
		// createIndex := esapi.IndicesCreate{
		// 	Index: s.Index,
		// }.Do(ctx, client)

		createIndex, err := client.Indices.Create(
			s.Index,
			client.Indices.Create.WithBody(nil),
		)

		// createIndex, err := client.Indices.Create(s.Index).Do()
		if err != nil {
			log.Fatal(err)
		} else {
			log.Infof("Created Index %v", s.Index)
		}

		if createIndex.IsError() {
			log.Fatalln("Could not create the Elastic index.. Exiting")
		}

	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         s.Index,         // The default index name
		Client:        client,          // The Elasticsearch client
		NumWorkers:    8,               // The number of worker goroutines
		FlushBytes:    int(5e+6),       // The flush threshold in bytes
		FlushInterval: 4 * time.Second, // The periodic flush interval
	})
	if err != nil {
		log.Fatalf("Error creating the indexer: %s", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 640*1024)
	scanner.Buffer(buf, 1024*1024*10)

	cnt := 0
	for scanner.Scan() {
		err = bi.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				// Action field configures the operation to perform (index, create, delete, update)
				Action: "index",

				// DocumentID is the (optional) document ID
				// DocumentID: strconv.Itoa(a.ID),

				// Body is an `io.Reader` with the payload
				Body: bytes.NewReader([]byte(scanner.Text())),

				// OnSuccess is called for each successful operation
				// OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
				// 	atomic.AddUint64(&countSuccessful, 1)
				// },

				// OnFailure is called for each failed operation
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						log.Printf("ERROR: %s", err)
					} else {
						log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
					}
				},
			},
		)

		cnt++
		if cnt%1000 == 0 {
			log.Infoln(cnt)
		}

	}
	if err := bi.Close(context.Background()); err != nil {
		log.Fatalf("Unexpected error: %s", err)
	}

	time.Sleep(5 * time.Second)

}
