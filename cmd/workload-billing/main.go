package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	greeting := os.Getenv("GREETING")
	if greeting == "" {
		greeting = "Hello from billing service"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s (port %s)\n", greeting, port)
	})

	http.HandleFunc("/billing/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s (port %s)\n", greeting, port)
	})

	log.Printf("Billing service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
