package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Service represents a network endpoint to monitor
type Service struct {
	Name     string
	URL      string
	Insecure bool
}

// ServiceResult holds the check result for a service
type ServiceResult struct {
	Service      Service
	Online       bool
	ResponseTime time.Duration
}

// checkService checks a service and measures response time
func checkService(url string, insecure bool) (bool, time.Duration) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	start := time.Now()
	_, err := client.Get(url)
	elapsed := time.Since(start)

	if err != nil {
		return false, 0
	}
	return true, elapsed
}

// checkServiceConcurrent checks a service and sends result to a channel
func checkServiceConcurrent(svc Service, results chan<- ServiceResult, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Checking %s at %s...\n", svc.Name, svc.URL)

	online, responseTime := checkService(svc.URL, svc.Insecure)

	result := ServiceResult{
		Service:      svc,
		Online:       online,
		ResponseTime: responseTime,
	}

	results <- result
}

// printResult displays a service check result
func printResult(result ServiceResult) {
	status := "DOWN"
	if result.Online {
		status = "UP"
	}

	if result.Online {
		// Format with 2 decimal places for seconds
		seconds := result.ResponseTime.Seconds()
		fmt.Printf("[%s] %-30s Response time: %.2fs\n", status, result.Service.Name, seconds)
	} else {
		fmt.Printf("[%s] %-30s (No response)\n", status, result.Service.Name)
	}
}

func main() {
	fmt.Println("=== Network Health Monitor ===")

	// Define all services to monitor
	services := []Service{
		{Name: "Home Router", URL: "https://192.168.3.1", Insecure: true},
		{Name: "Google", URL: "https://www.google.com", Insecure: false},
		{Name: "GitHub", URL: "https://github.com", Insecure: false},
		{Name: "Cape Town, SA (UCT)", URL: "https://www.uct.ac.za", Insecure: false},
	}

	// Create a channel to receive results
	results := make(chan ServiceResult, len(services))

	// WaitGroup to track goroutines
	var wg sync.WaitGroup

	// Record start time
	startTime := time.Now()

	// Launch a goroutine for each service check
	for _, service := range services {
		wg.Add(1)
		go checkServiceConcurrent(service, results, &wg)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Collect and print all results
	fmt.Println("\n=== Results ===")
	for result := range results {
		printResult(result)
	}

	// Show total execution time
	elapsed := time.Since(startTime)
	fmt.Printf("\n=== Summary ===")
	fmt.Printf("\nTotal services checked: %d", len(services))
	fmt.Printf("\nTotal execution time: %.2fs\n", elapsed.Seconds())
}
