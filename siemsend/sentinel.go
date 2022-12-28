package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func (s Sentinel) Init() error {
	sentinelCmd := &cobra.Command{
		Use:   "sentinel [arguments]",
		Short: "send input data to Microsoft Sentinel",
		Long:  `make sure your data is in jsonl format, meaning each line is a separate json object. They'll be reconstructed into a JSON array and sent to Microsoft Sentinel.`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			s.Send(cmd, args)
		},
	}
	flags := sentinelCmd.Flags()
	flags.StringVarP(&s.CustomerId, "customer_id", "", "", "customer id")
	flags.StringVarP(&s.SharedKey, "shared_key", "", "", "shared key")
	flags.StringVarP(&s.LogType, "log_type", "", "", "log type")
	flags.UintVarP(&s.BatchSize, "batch_size", "", 100, "batch size")
	flags.StringVarP(&s.Proxy, "proxy", "", "", "proxy url")
	flags.BoolVarP(&s.Compression, "compression", "", false, "compression")
	rootCmd.AddCommand(sentinelCmd)
	return nil
}

type Sentinel struct {
	CustomerId  string
	SharedKey   string
	LogType     string
	BatchSize   uint
	Proxy       string
	Compression bool
}

type SignatureElements struct {
	Date          string // in rfc1123date format ('%a, %d %b %Y %H:%M:%S GMT')
	ContentLength uint
	Method        string
	ContentType   string
	Resource      string
}

// initializing and starting the sentinel object means the flags will be populated just in time
var _ = Sentinel{}.Init()

func (s Sentinel) buildSignature(sig SignatureElements) (string, error) {
	// build HMAC signature
	tmpl, err := template.New("sign").Parse(`{{.Method}}
{{.ContentLength}}
{{.ContentType}}
x-ms-date:{{.Date}}
{{.Resource}}`)

	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, sig); err != nil {
		return "", err
	}
	sharedKeyBytes, err := base64.StdEncoding.DecodeString(s.SharedKey)
	if err != nil {
		return "", err
	}
	h := hmac.New(sha256.New, []byte(sharedKeyBytes))
	h.Write(buf.Bytes())
	signature := fmt.Sprintf("SharedKey %s:%s", s.CustomerId, base64.StdEncoding.EncodeToString(h.Sum(nil)))
	return signature, nil
}

func (s Sentinel) sendBatch(batch string, totalSize uint) {
	// send batch to Microsoft Sentinel
	// build signature
	location, _ := time.LoadLocation("GMT")
	signatureElemets := SignatureElements{
		Date:          time.Now().In(location).Format(time.RFC1123),
		Method:        "POST",
		ContentLength: totalSize,
		ContentType:   "application/json",
		Resource:      "/api/logs",
	}
	signature, err := s.buildSignature(signatureElemets)
	if err != nil {
		PrintBatch(batch)
		log.Error(err)
		return
	}
	// build request
	uri := "https://" + s.CustomerId + ".ods.opinsights.azure.com" + signatureElemets.Resource + "?api-version=2016-04-01"
	headers := map[string]string{
		"x-ms-date":     signatureElemets.Date,
		"content-type":  signatureElemets.ContentType,
		"Authorization": signature,
		"Log-Type":      s.LogType,
	}
	// send request
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer([]byte(batch)))
	if err != nil {
		PrintBatch(batch)
		log.Error(err)
		return
	}
	var res *http.Response

	for k, v := range headers {
		req.Header[k] = []string{v}
	}
	if s.Proxy != "" {
		proxyURL, err := url.Parse(s.Proxy)
		if err != nil {
			PrintBatch(batch)
			log.Error(err)
			return
		}
		client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
		res, err = client.Do(req)
		if err != nil {
			PrintBatch(batch)
			log.Error(err)
			return
		}
	} else {
		// command, _ := http2curl.GetCurlCommand(req)
		// fmt.Println(command)
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			PrintBatch(batch)
			log.Error(err)
			return
		}
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		log.Infof("batch sent, with code %d", res.StatusCode)
	} else {
		log.Errorf("batch not sent, with code %d", res.StatusCode)
		PrintBatch(batch)
	}

}

// this is used when an error accours, and we want to send the batch to stdout
func PrintBatch(batch string) {
	batch = strings.TrimSuffix(batch, "]")
	batch = strings.TrimPrefix(batch, "[")
	batch = strings.ReplaceAll(batch, "},{", "}\n{")
	// send the failed batch to stdout
	fmt.Printf("%s\n", batch)
}

func (s Sentinel) Send(cmd *cobra.Command, args []string) {
	// todo: build the send function and for loop
	batch := "["
	cnt := 0

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if scanner.Scan() {
			cnt++
			batch += scanner.Text()
			batch += ","
			if cnt == int(s.BatchSize) {
				// remove the last ,
				batch = strings.TrimSuffix(batch, ",")
				batch += "]"
				s.sendBatch(batch, uint(len(batch)))
				//reset counters
				batch = "["
				cnt = 0
			}
		} else {
			if batch != "[" {
				batch = strings.TrimSuffix(batch, ",")
				batch += "]"
				s.sendBatch(batch, uint(len(batch)))
			}
			break
		}
	}
}
