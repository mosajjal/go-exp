package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"moul.io/http2curl"
)

func handler(w http.ResponseWriter, r *http.Request) {
	command, _ := http2curl.GetCurlCommand(r)
	fmt.Println(command)
}

func errorHandler(err error) {
	if err != nil {
		log.Fatal("fatal Error: ", err)
	}
}

func main() {

	address := flag.String("address", "127.0.0.1", "Bind address")
	port := flag.Uint("port", 8080, "listen port")
	path := flag.String("path", "/", "accept requests on this path")
	mode := flag.String("mode", "http", "server type to use. options: http, tls.")
	tlsCert := flag.String("tlsCert", "", "tls certificate to use. will use self-signed if empty")
	tlsKey := flag.String("tlsKey", "", "tls certificate key to use. will use self-signed if empty")

	flag.Parse()
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)

	if (*tlsCert == "" || *tlsKey == "") && *mode == "tls" {
		cert, key, err := GenerateSelfSignedCertKey(*address, nil, nil)
		errorHandler(err)

		certFile, err := ioutil.TempFile("/tmp", "spitcurl-cert")
		errorHandler(err)
		defer os.Remove(certFile.Name())
		certFile.Write(cert)
		*tlsCert = certFile.Name()

		keyFile, err := ioutil.TempFile("/tmp", "spitcurl-key")
		errorHandler(err)
		defer os.Remove(keyFile.Name())
		keyFile.Write(key)
		*tlsKey = keyFile.Name()

	}

	http.HandleFunc(*path, handler)
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
