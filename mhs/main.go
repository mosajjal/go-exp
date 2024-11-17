// Package main is the main package for the HEC server
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

var (
	validToken string
	verbose    bool
)

// Event is a struct that represents the event that is sent to the HEC
type Event struct {
	Event any `json:"event"`
}

func main() {
	flag.StringVar(&validToken, "token", "00000000-0000-0000-0000-000000000000", "Token for authentication")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/services/collector", tokenAuth(eventHandler))
	mux.HandleFunc("/services/collector/event", tokenAuth(eventHandler))
	mux.HandleFunc("/services/collector/event/1.0", tokenAuth(eventHandler))
	mux.HandleFunc("/services/collector/event/1.0/raw", tokenAuth(rawEventHandler))
	mux.HandleFunc("/services/collector/event/1.0/validate", tokenAuth(validateEventHandler))
	mux.HandleFunc("/services/collector/event/1.0/validate_bulk", tokenAuth(validateBulkEventHandler))
	mux.HandleFunc("/services/collector/health/1.0", healthHandler)

	log.Println("Starting server on :8088")
	if err := http.ListenAndServe(":8088", log404Handler(mux)); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

func tokenAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token != "Splunk "+validToken && token != validToken {
			http.Error(w, "Forbidden", http.StatusForbidden)
			logResponse(w, http.StatusForbidden, "Forbidden")
			return
		}
		next.ServeHTTP(w, r)
	}
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		logResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// we expect the body to be at least one line, and each line containing a separate event that needs to be handled

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer r.Body.Close()

	// split the body into lines
	lines := bytes.Split(body, []byte("\n"))

	// for each line, parse the event and handle it
	for _, line := range lines {
		// for empty lines, skip
		if len(line) == 0 || string(line) == "\n" || string(line) == "\r\n" || string(line) == "\r" {
			continue
		}
		// this method does not support events taht are not delimited by a newline

		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			logResponse(w, http.StatusBadRequest, "Bad request")
			return
		}

		log.Printf("Received event: %s\n", event.Event)
	}
	w.WriteHeader(http.StatusOK)
	response := `{"text":"Success","code":0}`
	fmt.Fprintln(w, response)
	logResponse(w, http.StatusOK, response)
}

func rawEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		logResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer r.Body.Close()

	log.Printf("Received raw event: %s\n", string(body))
	w.WriteHeader(http.StatusOK)
	response := `{"text":"Success","code":0}`
	fmt.Fprintln(w, response)
	logResponse(w, http.StatusOK, response)
}

func validateEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		logResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer r.Body.Close()

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		logResponse(w, http.StatusBadRequest, "Bad request")
		return
	}

	log.Printf("Validated event: %s\n", event.Event)
	w.WriteHeader(http.StatusOK)
	response := `{"text":"Validation successful","code":0}`
	fmt.Fprintln(w, response)
	logResponse(w, http.StatusOK, response)
}

func validateBulkEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		logResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer r.Body.Close()

	var events []Event
	if err := json.Unmarshal(body, &events); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		logResponse(w, http.StatusBadRequest, "Bad request")
		return
	}

	for _, event := range events {
		log.Printf("Validated bulk event: %s\n", event.Event)
	}
	w.WriteHeader(http.StatusOK)
	response := `{"text":"Bulk validation successful","code":0}`
	fmt.Fprintln(w, response)
	logResponse(w, http.StatusOK, response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		logResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	log.Println("Health check")
	w.WriteHeader(http.StatusOK)
	response := `{"text":"HEC is healthy","code":0}`
	fmt.Fprintln(w, response)
	logResponse(w, http.StatusOK, response)
}

func logRequest(r *http.Request) {
	log.Printf("Request: %s %s\n", r.Method, r.URL)
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("Header: %s=%s\n", name, value)
		}
	}
	if verbose {
		body, _ := io.ReadAll(r.Body)
		log.Printf("Body: %s\n", string(body))
		r.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset body for further use
	}
}

func logResponse(_ http.ResponseWriter, statusCode int, response string) {
	log.Printf("Response: %d %s\n", statusCode, response)
	if verbose {
		log.Printf("Response Body: %s\n", response)
	}
}

func log404Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.statusCode == http.StatusNotFound {
			log.Printf("404 Not Found: %s %s\n", r.Method, r.URL)
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}
