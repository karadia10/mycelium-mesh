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
		port = "8082"
	}

	greeting := os.Getenv("GREETING")
	if greeting == "" {
		greeting = "Hello from frontend service"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s (port %s)\n", greeting, port)
	})

	http.HandleFunc("/frontend/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s (port %s)\n", greeting, port)
	})

	log.Printf("Frontend service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
