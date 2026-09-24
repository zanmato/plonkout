// Command healthchecker exits 0 when the URL answers 200. The runtime image has
// no shell or curl, so the container health check runs this instead.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	url := "http://127.0.0.1:8080/api/health"
	if len(os.Args) > 1 && os.Args[1] != "" {
		url = os.Args[1]
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "%s answered %d\n", url, resp.StatusCode)
		os.Exit(1)
	}
}
