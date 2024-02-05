package main

import (
	"crypto/tls"
	"flag"
	"fmt"
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

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}
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
		certFile, err := os.CreateTemp(os.TempDir(), "spitcurl.pem")
		if err != nil {
			log.Fatal("fatal Error: ", err)
		}
		defer os.Remove(certFile.Name())
		certFile.Write(cert)
		*tlsCert = certFile.Name()

		keyFile, err := os.CreateTemp(os.TempDir(), "spitcurl.key")
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
			if len(keyValue) != 2 {
				continue
			}
			w.Header().Add(keyValue[0], keyValue[1])
		}
		w.WriteHeader(200)
		w.Write([]byte(command.String()))
	}

	if *upstream == "" {
		http.HandleFunc("/", handlerEmpty)
	} else {
		// proxy := httputil.NewSingleHostReverseProxy(remote)

		director := func(req *http.Request) {
			targetQuery := remote.RawQuery
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path, req.URL.RawPath = joinURLPath(remote, req.URL)
			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
		}
		// skip tls verification on upstream
		proxy := &httputil.ReverseProxy{Director: director,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

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
