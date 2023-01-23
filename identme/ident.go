package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/netip"
)

func main() {

	l := flag.String("l", ":8080", "listen address")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ipport, _ := netip.ParseAddrPort(r.RemoteAddr)
		fmt.Fprint(w, ipport.Addr().String())
	})

	http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		ipport, _ := netip.ParseAddrPort(r.RemoteAddr)
		j, _ := json.Marshal(map[string]string{"ip": ipport.Addr().String()})
		fmt.Fprint(w, string(j))
	})
	log.Println("Listening on", *l)
	log.Fatalln(http.ListenAndServe(*l, nil))
}
