package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// Service represents a network endpoint to monitor
type Service struct {
	Name     string
	URL      string
	Insecure bool // true for local devices with self-signed certs
}

// checkHTTPS checks an HTTPS endpoint
func checkHTTPS(url string, insecure bool) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// If insecure, skip certificate verification
	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	_, err := client.Get(url)
	return err == nil
}

// checkLatency measures response time to a URL
func checkLatency(url string) (bool, time.Duration) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	start := time.Now()
	_, err := client.Get(url)
	elapsed := time.Since(start)

	if err != nil {
		return false, 0
	}
	return true, elapsed
}

// checkService tests a service and prints results
func checkService(svc Service, measureLatency bool) {
	fmt.Printf("Checking %s at %s...\n", svc.Name, svc.URL)

	if measureLatency {
		online, latency := checkLatency(svc.URL)
		status := "DOWN"
		if online {
			status = "UP"
		}
		fmt.Printf("[%s] %s - Latency: %v\n\n", status, svc.Name, latency)
	} else {
		online := checkHTTPS(svc.URL, svc.Insecure)
		status := "DOWN"
		if online {
			status = "UP"
		}
		fmt.Printf("[%s] %s\n\n", status, svc.Name)
	}
}

func main() {
	fmt.Println("=== Network Health Monitor ===")

	// Define all services to monitor in a slice
	services := []Service{
		{Name: "Home Router", URL: "https://192.168.3.1", Insecure: true},
		{Name: "Google", URL: "https://www.google.com", Insecure: false},
		{Name: "GitHub", URL: "https://github.com", Insecure: false},
		{Name: "Cape Town, SA (UCT)", URL: "https://www.uct.ac.za", Insecure: false},
	}

	// Check each service
	for _, service := range services {
		// Measure latency only for Cape Town
		measureLatency := service.Name == "Cape Town, SA (UCT)"
		checkService(service, measureLatency)
	}
}
