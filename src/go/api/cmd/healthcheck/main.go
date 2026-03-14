package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	url := os.Getenv("HEALTHCHECK_URL")
	if url == "" {
		url = "http://localhost:8090/healthz"
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: status %d\n", resp.StatusCode)
		os.Exit(1)
	}
}
