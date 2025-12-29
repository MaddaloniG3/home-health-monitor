package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorBlue   = "\033[34m"
)

// ServiceType defines the type of check to perform
type ServiceType string

const (
	TypeHTTP ServiceType = "http"
	TypePing ServiceType = "ping"
	TypeDNS  ServiceType = "dns"
)

// Service represents a network endpoint to monitor
type Service struct {
	Name     string
	URL      string
	Host     string
	Type     ServiceType
	Insecure bool
	Location string // Geographic location for latency tests
}

// ServiceResult holds the check result for a service
type ServiceResult struct {
	Service      Service
	Online       bool
	ResponseTime time.Duration
	Error        string
	Timestamp    time.Time
}

// checkHTTP checks an HTTP/HTTPS endpoint
func checkHTTP(url string, insecure bool) (bool, time.Duration, string) {
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
		return false, 0, err.Error()
	}
	return true, elapsed, ""
}

// checkPing performs a ping using the system ping command
func checkPing(host string) (bool, time.Duration, string) {
	cmd := exec.Command("ping", "-c", "3", "-W", "5000", host)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	if err != nil {
		return false, 0, "Host unreachable"
	}

	// Parse average ping time from output
	re := regexp.MustCompile(`avg = ([\d.]+)`)
	matches := re.FindStringSubmatch(string(output))

	if len(matches) > 1 {
		avgMs, _ := strconv.ParseFloat(matches[1], 64)
		avgTime := time.Duration(avgMs * float64(time.Millisecond))
		return true, avgTime, ""
	}

	return true, elapsed / 3, ""
}

// checkDNS performs a DNS lookup and measures response time
func checkDNS(host string) (bool, time.Duration, string) {
	resolver := &net.Resolver{}

	start := time.Now()
	_, err := resolver.LookupHost(context.Background(), host)
	elapsed := time.Since(start)

	if err != nil {
		return false, 0, err.Error()
	}
	return true, elapsed, ""
}

// checkServiceConcurrent checks a service and sends result to a channel
func checkServiceConcurrent(svc Service, results chan<- ServiceResult, wg *sync.WaitGroup) {
	defer wg.Done()

	var online bool
	var responseTime time.Duration
	var errMsg string

	switch svc.Type {
	case TypeHTTP:
		online, responseTime, errMsg = checkHTTP(svc.URL, svc.Insecure)
	case TypePing:
		online, responseTime, errMsg = checkPing(svc.Host)
	case TypeDNS:
		online, responseTime, errMsg = checkDNS(svc.Host)
	default:
		errMsg = "Unknown service type"
	}

	result := ServiceResult{
		Service:      svc,
		Online:       online,
		ResponseTime: responseTime,
		Error:        errMsg,
		Timestamp:    time.Now(),
	}

	results <- result
}

// printResult displays a service check result with colors
func printResult(result ServiceResult) {
	timestamp := result.Timestamp.Format("15:04:05")
	checkType := string(result.Service.Type)

	var statusColor string
	var status string

	if result.Online {
		statusColor = ColorGreen
		status = "UP"
	} else {
		statusColor = ColorRed
		status = "DOWN"
	}

	// Add location info if present
	nameWithLocation := result.Service.Name
	if result.Service.Location != "" {
		nameWithLocation = fmt.Sprintf("%s (%s)", result.Service.Name, result.Service.Location)
	}

	if result.Online {
		seconds := result.ResponseTime.Seconds()
		ms := result.ResponseTime.Milliseconds()

		// Use milliseconds for fast responses (< 1 second)
		if seconds < 1.0 {
			fmt.Printf("%s[%s]%s [%s] %-45s [%s] %3dms\n",
				statusColor, status, ColorReset, timestamp,
				nameWithLocation, checkType, ms)
		} else {
			fmt.Printf("%s[%s]%s [%s] %-45s [%s] %.2fs\n",
				statusColor, status, ColorReset, timestamp,
				nameWithLocation, checkType, seconds)
		}
	} else {
		fmt.Printf("%s[%s]%s [%s] %-45s [%s] %s\n",
			statusColor, status, ColorReset, timestamp,
			nameWithLocation, checkType, result.Error)
	}
}

// writeToLog appends result to log file
func writeToLog(result ServiceResult, logFile *os.File) {
	if logFile == nil {
		return
	}

	timestamp := result.Timestamp.Format("2006-01-02 15:04:05")
	status := "UP"
	if !result.Online {
		status = "DOWN"
	}

	nameWithLocation := result.Service.Name
	if result.Service.Location != "" {
		nameWithLocation = fmt.Sprintf("%s (%s)", result.Service.Name, result.Service.Location)
	}

	logLine := fmt.Sprintf("%s | [%s] %-45s | Type: %s | Response: %dms",
		timestamp, status, nameWithLocation,
		result.Service.Type, result.ResponseTime.Milliseconds())

	if !result.Online {
		logLine += fmt.Sprintf(" | Error: %s", result.Error)
	}

	logLine += "\n"
	logFile.WriteString(logLine)
}

// runHealthCheck performs one complete health check cycle
func runHealthCheck(services []Service, logFile *os.File) {
	results := make(chan ServiceResult, len(services))
	var wg sync.WaitGroup

	startTime := time.Now()

	for _, service := range services {
		wg.Add(1)
		go checkServiceConcurrent(service, results, &wg)
	}

	wg.Wait()
	close(results)

	// Collect results and organize by category
	var localResults []ServiceResult
	var webResults []ServiceResult
	var latencyResults []ServiceResult

	successCount := 0
	var totalResponseTime time.Duration

	for result := range results {
		if result.Service.Type == TypeHTTP && result.Service.Location == "" {
			webResults = append(webResults, result)
		} else if result.Service.Type == TypePing && result.Service.Location == "" {
			localResults = append(localResults, result)
		} else if result.Service.Location != "" {
			latencyResults = append(latencyResults, result)
		}

		if result.Online {
			successCount++
			totalResponseTime += result.ResponseTime
		}
	}

	// Print organized results
	if len(localResults) > 0 {
		fmt.Printf("\n%s=== Local Network ===%s\n", ColorBlue, ColorReset)
		for _, result := range localResults {
			printResult(result)
			writeToLog(result, logFile)
		}
	}

	if len(webResults) > 0 {
		fmt.Printf("\n%s=== Web Services ===%s\n", ColorBlue, ColorReset)
		for _, result := range webResults {
			printResult(result)
			writeToLog(result, logFile)
		}
	}

	if len(latencyResults) > 0 {
		fmt.Printf("\n%s=== Global Latency Tests ===%s\n", ColorBlue, ColorReset)
		for _, result := range latencyResults {
			printResult(result)
			writeToLog(result, logFile)
		}
	}

	// Calculate statistics
	elapsed := time.Since(startTime)
	successRate := float64(successCount) / float64(len(services)) * 100
	avgResponseTime := time.Duration(0)
	if successCount > 0 {
		avgResponseTime = totalResponseTime / time.Duration(successCount)
	}

	// Print summary
	fmt.Printf("\n%s=== Summary ===%s", ColorCyan, ColorReset)
	fmt.Printf("\nTotal services checked: %d", len(services))
	fmt.Printf("\n%sSuccess rate: %.1f%% (%d/%d)%s",
		ColorGreen, successRate, successCount, len(services), ColorReset)
	fmt.Printf("\nAverage response time: %dms", avgResponseTime.Milliseconds())
	fmt.Printf("\nTotal execution time: %.2fs\n", elapsed.Seconds())
}

func main() {
	fmt.Printf("%s=== Network Health Monitor ===%s\n", ColorCyan, ColorReset)
	fmt.Println("Press Ctrl+C to stop monitoring")

	// Open log file
	logFile, err := os.OpenFile("health_monitor.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("%sWarning: Could not open log file: %v%s\n",
			ColorYellow, err, ColorReset)
		logFile = nil
	} else {
		defer logFile.Close()
		fmt.Printf("%sLogging to: health_monitor.log%s\n\n", ColorYellow, ColorReset)
	}

	// Define all services to monitor
	services := []Service{
		// Local Network
		{Name: "Home Router", URL: "https://192.168.3.1", Type: TypeHTTP, Insecure: true},
		{Name: "Router", Host: "192.168.3.1", Type: TypePing},

		// Web Services
		{Name: "Google", URL: "https://www.google.com", Type: TypeHTTP},
		{Name: "GitHub", URL: "https://github.com", Type: TypeHTTP},
		{Name: "Mastercard Website", URL: "https://www.mastercard.com", Type: TypeHTTP},
		{Name: "George Maddaloni Website", URL: "https://www.georgemaddaloni.com", Type: TypeHTTP},
		{Name: "MA Connect Website", URL: "https://www.mastercardconnect.com", Type: TypeHTTP},

		// Global Latency Tests - Using real HTTP endpoints in each region

		// Africa
		{Name: "Johannesburg", URL: "https://www.takealot.com", Type: TypeHTTP, Location: "South Africa"},
		{Name: "Cape Town", URL: "https://www.uct.ac.za", Type: TypeHTTP, Location: "South Africa"},

		// South America
		{Name: "São Paulo", URL: "https://www.uol.com.br", Type: TypeHTTP, Location: "Brazil"},
		{Name: "Mexico City", URL: "https://www.mercadolibre.com.mx", Type: TypeHTTP, Location: "Mexico"},

		// Europe
		{Name: "Paris", URL: "https://www.lemonde.fr", Type: TypeHTTP, Location: "France"},
		{Name: "Frankfurt", URL: "https://www.bundesregierung.de", Type: TypeHTTP, Location: "Germany"},
		{Name: "London", URL: "https://www.bbc.com", Type: TypeHTTP, Location: "UK"},

		// Middle East
		{Name: "Dubai", URL: "https://www.emirates.com", Type: TypeHTTP, Location: "UAE"},
		{Name: "Riyadh", URL: "https://www.spa.gov.sa", Type: TypeHTTP, Location: "Saudi Arabia"},

		// Asia - South
		{Name: "Mumbai", URL: "https://www.timesofindia.com", Type: TypeHTTP, Location: "India"},
		{Name: "Pune", URL: "https://www.infosys.com", Type: TypeHTTP, Location: "India"},

		// Asia - Southeast
		{Name: "Singapore", URL: "https://www.straitstimes.com", Type: TypeHTTP, Location: "Singapore"},
		{Name: "Bangkok", URL: "https://www.sanook.com", Type: TypeHTTP, Location: "Thailand"},

		// Asia - East
		{Name: "Tokyo", URL: "https://www.yahoo.co.jp", Type: TypeHTTP, Location: "Japan"},

		// Oceania
		{Name: "Sydney", URL: "https://www.abc.net.au", Type: TypeHTTP, Location: "Australia"},
		{Name: "Melbourne", URL: "https://www.theage.com.au", Type: TypeHTTP, Location: "Australia"},

		// North America
		{Name: "Ashburn", URL: "https://aws.amazon.com", Type: TypeHTTP, Location: "Virginia, USA"},
		{Name: "Dallas", URL: "https://www.att.com", Type: TypeHTTP, Location: "Texas, USA"},
		{Name: "San Jose", URL: "https://www.cisco.com", Type: TypeHTTP, Location: "California, USA"},
	}

	interval := 30 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("%s[%s] Starting health check cycle...%s\n",
		ColorCyan, time.Now().Format("15:04:05"), ColorReset)
	runHealthCheck(services, logFile)

	for range ticker.C {
		fmt.Printf("\n%s[%s] Starting health check cycle...%s\n",
			ColorCyan, time.Now().Format("15:04:05"), ColorReset)
		runHealthCheck(services, logFile)
	}
}
