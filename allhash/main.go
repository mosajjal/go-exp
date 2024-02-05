// allhash binary generates all Hashes for a binary that VT provides.
package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/glaslos/ssdeep"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"

	"github.com/glaslos/tlsh"

	"github.com/spf13/pflag"
)

type hash struct {
	Name    string `yaml:"alg" json:"alg"`
	sum     []byte
	Encoded string `yaml:"hash" json:"hash"`
}

type config struct {
	format   string
	encoding string
	Input    string
	Hashes   []hash
}

func (c config) encoder(i []byte) (r string) {
	switch c.encoding {
	case "hex":
		r = fmt.Sprintf("%x", i)
	case "base32":
		r = base32.StdEncoding.EncodeToString(i)
	case "base64":
		r = base64.StdEncoding.EncodeToString(i)
	}
	return
}

func (c config) marshaller() (r string) {
	switch c.format {
	case "yaml", "pretty":
		if m, err := yaml.Marshal(c); err == nil {
			r = string(m)
		} else {
			glog.Error(err)
		}

	case "json":
		if m, err := json.Marshal(c); err == nil {
			r = string(m)
		} else {
			glog.Error(err)
		}

	}
	return
}

func main() {
	c := config{}

	pflag.StringVarP(&c.format, "format", "f", "pretty", "formatting, choices: yaml, json, pretty")
	pflag.StringVarP(&c.encoding, "encoding", "e", "hex", "encoding, choices: hex, base32, base64")
	pflag.StringVarP(&c.Input, "input", "i", "", "input file path") // todo: add glob and directory support

	pflag.Parse()

	// verify flags
	if c.format != "yaml" && c.format != "json" && c.format != "pretty" {
		glog.Fatal("invalid format")
	}
	if c.encoding != "hex" && c.encoding != "base32" && c.encoding != "base64" {
		glog.Fatal("invalid encoding")
	}

	f, err := os.Open(c.Input)
	if err != nil {
		glog.Fatal(err.Error())
	}
	defer f.Close()

	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {
		defer wg.Done()
		h := md5.New()
		if _, err := io.Copy(h, f); err != nil {
			glog.Fatal(err.Error())
		}
		c.Hashes = append(c.Hashes, hash{
			Name:    "md5",
			sum:     h.Sum(nil),
			Encoded: c.encoder(h.Sum(nil)),
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			glog.Fatal(err)
		}
		c.Hashes = append(c.Hashes, hash{
			Name:    "sha1",
			sum:     h.Sum(nil),
			Encoded: c.encoder(h.Sum(nil)),
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			glog.Fatal(err)
		}
		c.Hashes = append(c.Hashes, hash{
			Name:    "sha256",
			sum:     h.Sum(nil),
			Encoded: c.encoder(h.Sum(nil)),
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if t, e := tlsh.HashFilename(c.Input); e == nil {
			c.Hashes = append(c.Hashes, hash{
				Name:    "tlsh",
				sum:     t.Binary(),
				Encoded: c.encoder(t.Binary()),
			})
		} else {
			glog.Error(e)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if h, e := ssdeep.FuzzyFilename(c.Input); e == nil {
			c.Hashes = append(c.Hashes, hash{
				Name:    "ssdeep",
				sum:     nil,
				Encoded: h,
			})
		} else {
			glog.Warningf("ssdeep didn't compute: %s", e)
		}
	}()
	//todo: Telfhash and imphash
	wg.Wait()

	fmt.Println(c.marshaller())
}
