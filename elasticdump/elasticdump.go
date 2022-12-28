package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tidwall/gjson"
)

var targetIP = flag.String("targetIP", "", "Target IP Address")
var targetPort = flag.Uint("targetPort", 9200, "Target port")
var minDocCount = flag.Uint("minDocCount", 100, "Minimum number of Documents for each index")
var minIndexSizeKB = flag.Uint("minIndexSizeKB", 1024, "Minimum size of index for dump")
var indexRegex = flag.String("indexRegex", ".*", "Only download indices matching regex")

func checkFlags() {
	flag.Parse()
	if *targetPort > 65535 {
		log.Fatal("-targetPort must be between 1 and 65535")
	}

	if *targetIP == "" {
		log.Fatal("-targetIP is required")
	}
}

func check(e error) {
	if e != nil {
		log.Panic(e)
		// panic(e)
	}
}

func getNextScroll(ip string, port uint, scroll string, f *os.File) int {
	postData := []byte(fmt.Sprintf(`{"scroll_id": "%v", "scroll": "10m"}`, scroll))
	resp, err := http.Post(fmt.Sprintf("http://%s:%d/_search/scroll", ip, port), "application/json", bytes.NewBuffer(postData))
	check(err)
	defer resp.Body.Close()
	resBytes, _ := ioutil.ReadAll(resp.Body)
	Hits := gjson.GetBytes(resBytes, "hits.hits")
	if Hits.Raw == "" || Hits.Raw == "[]" {
		return 0
	}
	f.Write([]byte(Hits.Raw))
	nextScroll := gjson.GetBytes(resBytes, "_scroll_id")
	if nextScroll.Exists() {
		getNextScroll(ip, port, nextScroll.String(), f)
	}
	return 0
}

func indexToJSON(ip string, port uint, index string, done chan<- bool) (okay bool) {
	defer func() {
		done <- okay
	}()
	postData := []byte(`{"size": 1000}`)
	resp, err := http.Post(fmt.Sprintf("http://%s:%d/%v/_search?scroll=10m", ip, port, index), "application/json", bytes.NewBuffer(postData))
	check(err)
	_ = os.Mkdir(fmt.Sprintf("./%v/", ip), 0755)
	f, err := os.Create(fmt.Sprintf("./%v/ESDUMP-%v-%v-%v.json", ip, ip, index, time.Now().Format(time.RFC3339)))
	check(err)
	defer f.Close()
	defer resp.Body.Close()
	resBytes, _ := ioutil.ReadAll(resp.Body)
	Hits := gjson.GetBytes(resBytes, "hits.hits")
	if !(Hits.Raw == "" || Hits.Raw == "[]") {
		f.Write([]byte(Hits.Raw))
	}
	nextScroll := gjson.GetBytes(resBytes, "_scroll_id")
	if nextScroll.Exists() {
		_ = getNextScroll(ip, port, nextScroll.String(), f)

	}
	return true
}

func getIndexList(ip string, port uint, minDocCount uint, minIndexSizeKB uint, indexRe *regexp.Regexp) []string {

	var resList []string

	resp, err := http.Get(fmt.Sprintf("http://%s:%d/_cat/indices?format=json&bytes=kb", ip, port))
	check(err)
	defer resp.Body.Close()
	bytes, _ := ioutil.ReadAll(resp.Body)
	result := gjson.Parse(string(bytes))
	result.ForEach(func(key, value gjson.Result) bool {

		docCountInt, _ := strconv.Atoi(gjson.Get(value.String(), "docs\\.count").String())
		indexSizeKB, _ := strconv.Atoi(gjson.Get(value.String(), "store\\.size").String())
		if uint(docCountInt) >= minDocCount && uint(indexSizeKB) >= minIndexSizeKB {
			indexName := gjson.Get(value.String(), "index").String()
			// Checking index regex match
			if indexRe.Match([]byte(indexName)) {
				resList = append(resList, indexName)
			} else {
				log.Warnf("Index %s name did NOT match regex, skipping..", indexName)
			}
		} else {
			log.Warnf("Index %s name did NOT meet document count and size requirements, skipping..", gjson.Get(value.String(), "index").String())
		}
		return true
	})
	return resList
}

func main() {
	log.Info("Starting ...")
	checkFlags()
	indexRe := regexp.MustCompile(*indexRegex)
	log.Infof("Getting index list from %s", *targetIP)
	indexList := getIndexList(*targetIP, *targetPort, *minDocCount, *minIndexSizeKB, indexRe)
	done := make(chan bool)
	for _, index := range indexList {
		log.Infof("Getting index %s from %s", index, *targetIP)
		go indexToJSON(*targetIP, *targetPort, index, done)
	}
	// wait for everything to finish
	errors := 0
	for i := 0; i < len(indexList); i++ {
		if !<-done {
			log.Errorf("Error while doing index %s", indexList[i])
			errors++
		}
	}
}
