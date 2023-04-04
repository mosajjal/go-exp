package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"moul.io/http2curl/v2"
)

type headerFlags []string

func (i *headerFlags) String() string {
	// join by \n
	return strings.Join(*i, "\n")
}

func (i *headerFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var headerFlag headerFlags

func main() {
	address := flag.String("address", "127.0.0.1", "Bind address")
	port := flag.Uint("port", 8080, "listen port")
	upstream := flag.String("upstream", "", "upstream URL. Empty will return the curl for all requests, Example: https://www.youtube.com")
	mode := flag.String("mode", "http", "server type to use. options: http, tls.")
	tlsCert := flag.String("tlsCert", "", "tls certificate to use. will use self-signed if empty")
	tlsKey := flag.String("tlsKey", "", "tls certificate key to use. will use self-signed if empty")
	flag.Var(&headerFlag, "header", "headers to add to the response. Example: -header 'X-My-Header: my-value'")

	flag.Parse()
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)

	if (*tlsCert == "" || *tlsKey == "") && *mode == "tls" {
		cert, key, err := GenerateSelfSignedCertKey(*address, nil, nil)
		if err != nil {
			log.Fatal("fatal Error: ", err)
		}
		certFile, err := ioutil.TempFile(os.TempDir(), "spitcurl.pem")
		if err != nil {
			log.Fatal("fatal Error: ", err)
		}
		defer os.Remove(certFile.Name())
		certFile.Write(cert)
		*tlsCert = certFile.Name()

		keyFile, err := ioutil.TempFile(os.TempDir(), "spitcurl.key")
		if err != nil {
			log.Fatal("fatal Error: ", err)
		}
		defer os.Remove(keyFile.Name())
		keyFile.Write(key)
		*tlsKey = keyFile.Name()

	}

	remote, err := url.Parse(*upstream)
	if err != nil {
		panic(err)
	}
	handlerProxy := func(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			command, _ := http2curl.GetCurlCommand(r)
			fmt.Println(command.String())
			r.Host = remote.Host
			p.ServeHTTP(w, r)
		}
	}
	handlerEmpty := func(w http.ResponseWriter, r *http.Request) {
		command, _ := http2curl.GetCurlCommand(r)
		fmt.Println(command.String())
		// write the headers
		for _, header := range strings.Split(headerFlag.String(), "\n") {
			// split the header into key and value
			keyValue := strings.Split(header, ":")
			w.Header().Add(keyValue[0], keyValue[1])
		}
		w.WriteHeader(200)
		w.Write([]byte(command.String()))
	}

	if *upstream == "" {
		http.HandleFunc("/", handlerEmpty)
	} else {
		proxy := httputil.NewSingleHostReverseProxy(remote)
		http.HandleFunc("/", handlerProxy(proxy))
	}

	switch *mode {
	case "http":
		log.Debugf("starting HTTP server on %v:%d", *address, *port)
		log.Fatal(
			http.ListenAndServe(fmt.Sprintf("%v:%v", *address, *port), nil),
		)
	case "tls":
		log.Debugf("starting HTTPS server on %v:%d", *address, *port)
		log.Fatal(
			http.ListenAndServeTLS(fmt.Sprintf("%v:%v", *address, *port), *tlsCert, *tlsKey, nil),
		)
	}
}
