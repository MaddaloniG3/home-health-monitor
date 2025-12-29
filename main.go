package main

import (
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
)

// ServiceType defines the type of check to perform
type ServiceType string

const (
	TypeHTTP ServiceType = "http"
	TypePing ServiceType = "ping"
)

// Service represents a network endpoint to monitor
type Service struct {
	Name     string
	URL      string
	Host     string
	Type     ServiceType
	Insecure bool
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
		conn, dialErr := net.DialTimeout("tcp", host+":80", 3*time.Second)
		if dialErr == nil {
			conn.Close()
			return true, elapsed, ""
		}
		return false, 0, "Host unreachable"
	}

	re := regexp.MustCompile(`avg = ([\d.]+)`)
	matches := re.FindStringSubmatch(string(output))

	if len(matches) > 1 {
		avgMs, _ := strconv.ParseFloat(matches[1], 64)
		avgTime := time.Duration(avgMs * float64(time.Millisecond))
		return true, avgTime, ""
	}

	return true, elapsed / 3, ""
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

	if result.Online {
		seconds := result.ResponseTime.Seconds()
		fmt.Printf("%s[%s]%s [%s] %-30s [%s] Response time: %.2fs\n",
			statusColor, status, ColorReset, timestamp,
			result.Service.Name, checkType, seconds)
	} else {
		fmt.Printf("%s[%s]%s [%s] %-30s [%s] %s\n",
			statusColor, status, ColorReset, timestamp,
			result.Service.Name, checkType, result.Error)
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

	logLine := fmt.Sprintf("%s | [%s] %-30s | Type: %s | Response: %.2fs",
		timestamp, status, result.Service.Name,
		result.Service.Type, result.ResponseTime.Seconds())

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

	// Collect results and calculate statistics
	var allResults []ServiceResult
	var totalResponseTime time.Duration
	successCount := 0

	fmt.Println("\n=== Results ===")
	for result := range results {
		allResults = append(allResults, result)
		printResult(result)
		writeToLog(result, logFile)

		if result.Online {
			successCount++
			totalResponseTime += result.ResponseTime
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
	fmt.Printf("\n=== Summary ===")
	fmt.Printf("\nTotal services checked: %d", len(services))
	fmt.Printf("\n%sSuccess rate: %.1f%% (%d/%d)%s",
		ColorGreen, successRate, successCount, len(services), ColorReset)
	fmt.Printf("\nAverage response time: %.2fs", avgResponseTime.Seconds())
	fmt.Printf("\nTotal execution time: %.2fs\n", elapsed.Seconds())
}

func main() {
	fmt.Printf("%s=== Network Health Monitor ===%s\n", ColorCyan, ColorReset)
	fmt.Println("Press Ctrl+C to stop monitoring\n")

	// Open log file (optional - comment out if you don't want logging)
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
		{Name: "Home Router", URL: "https://192.168.3.1", Type: TypeHTTP, Insecure: true},
		{Name: "Google", URL: "https://www.google.com", Type: TypeHTTP, Insecure: false},
		{Name: "GitHub", URL: "https://github.com", Type: TypeHTTP, Insecure: false},
		{Name: "Mastercard Website", URL: "https://www.mastercard.com", Type: TypeHTTP, Insecure: false},
		{Name: "Cape Town, SA (UCT)", URL: "https://www.uct.ac.za", Type: TypeHTTP, Insecure: false},
		{Name: "George Maddaloni Website", URL: "https://www.georgemaddaloni.com", Type: TypeHTTP, Insecure: false},
		{Name: "MA Connect Website", URL: "https://www.mastercardconnect.com", Type: TypeHTTP, Insecure: false},
		{Name: "Router (ping)", Host: "192.168.3.1", Type: TypePing},
		{Name: "Google DNS", Host: "8.8.8.8", Type: TypePing},
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
