package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	// URL to send GET requests to
	url := "https://diablo-timer.paulgeorge.dev/api/subscription/push"

	// Calculate delay until next 5-minute mark
	now := time.Now()
	delay := time.Duration(5-now.Minute()%5) * time.Minute
	delay -= time.Duration(now.Second()) * time.Second
	delay -= time.Duration(now.Nanosecond()) * time.Nanosecond

	// Round the delay to seconds for cleaner output
	delaySeconds := delay.Round(time.Second)
	fmt.Printf("[%s] Waiting %v until first request\n", now.Format(time.RFC3339), delaySeconds)
	time.Sleep(delay)

	// Create a ticker that ticks every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Make initial request
	makeRequest(url)

	// Loop forever, making requests every 5 minutes
	for range ticker.C {
		makeRequest(url)
	}
}

func makeRequest(url string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("[%s] Error making request: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[%s] Error reading response: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}

	fmt.Printf("[%s] Response status: %s\n", time.Now().Format(time.RFC3339), resp.Status)
	fmt.Printf("[%s] Response body: %s\n", time.Now().Format(time.RFC3339), string(body))
}
