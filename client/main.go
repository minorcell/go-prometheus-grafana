package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var (
	baseURL     string
	concurrency int
	interval    time.Duration
)

func init() {
	flag.StringVar(&baseURL, "url", "http://localhost:8080", "Base URL of the server")
	flag.IntVar(&concurrency, "concurrency", 5, "Number of concurrent workers")
	flag.DurationVar(&interval, "interval", 500*time.Millisecond, "Request interval per worker")
}

var endpoints = []struct {
	path   string
	weight float64
}{
	{"/api/hello", 0.4},
	{"/api/slow", 0.1},
	{"/api/error", 0.2},
	{"/api/order", 0.2},
	{"/health", 0.1},
}

func pickEndpoint() string {
	r := rand.Float64()
	cumulative := 0.0
	for _, ep := range endpoints {
		cumulative += ep.weight
		if r <= cumulative {
			return ep.path
		}
	}
	return endpoints[0].path
}

func worker(id int, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{Timeout: 10 * time.Second}

	for {
		endpoint := pickEndpoint()
		url := baseURL + endpoint

		start := time.Now()
		resp, err := client.Get(url)
		duration := time.Since(start)

		if err != nil {
			log.Printf("[worker-%d] ERROR %s: %v", id, endpoint, err)
		} else {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			log.Printf("[worker-%d] %s %d %s", id, endpoint, resp.StatusCode, duration.Round(time.Millisecond))
		}

		jitter := time.Duration(rand.Int63n(int64(interval)))
		time.Sleep(interval/2 + jitter)
	}
}

func main() {
	flag.Parse()

	fmt.Printf("Load generator started\n")
	fmt.Printf("  Target: %s\n", baseURL)
	fmt.Printf("  Workers: %d\n", concurrency)
	fmt.Printf("  Interval: %s\n", interval)
	fmt.Println()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker(i, &wg)
	}
	wg.Wait()
}
