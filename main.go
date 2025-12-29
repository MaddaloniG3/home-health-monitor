package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	ColorReset   = "\033[0m"
	ColorGreen   = "\033[32m"
	ColorRed     = "\033[31m"
	ColorYellow  = "\033[33m"
	ColorCyan    = "\033[36m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
)

// TestType defines the type of test
type TestType string

const (
	TestTypePing TestType = "PING"
	TestTypeDNS  TestType = "DNS"
	TestTypeHTTP TestType = "HTTP"
)

// CloudEndpoint represents a cloud infrastructure endpoint
type CloudEndpoint struct {
	Location string
	Region   string
	Provider string // AWS, Azure, GCP
	Hostname string
	TestPing bool
	TestDNS  bool
	TestHTTP bool
}

// TestResult holds the result of a test
type TestResult struct {
	Endpoint     CloudEndpoint
	TestType     TestType
	Online       bool
	ResponseTime time.Duration
	ResolvedIP   string
	Error        string
	Timestamp    time.Time
	Trend        string
	Baseline     time.Duration
}

// HistoricalDataPoint represents a single measurement
type HistoricalDataPoint struct {
	Timestamp    time.Time
	ResponseTime time.Duration
}

// ServiceHistory tracks historical data for a service
type ServiceHistory struct {
	ServiceName string
	DataPoints  []HistoricalDataPoint
}

// HistoryStore manages all historical data
type HistoryStore struct {
	Services map[string]*ServiceHistory
	mu       sync.Mutex
}

// NewHistoryStore creates a new history store
func NewHistoryStore() *HistoryStore {
	return &HistoryStore{
		Services: make(map[string]*ServiceHistory),
	}
}

// LoadFromFile loads historical data from JSON file
func (hs *HistoryStore) LoadFromFile(filename string) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var rawData map[string][]struct {
		Timestamp    time.Time
		ResponseTime int64
	}

	if err := json.Unmarshal(data, &rawData); err != nil {
		return err
	}

	for serviceName, points := range rawData {
		history := &ServiceHistory{
			ServiceName: serviceName,
			DataPoints:  make([]HistoricalDataPoint, 0, len(points)),
		}

		for _, point := range points {
			history.DataPoints = append(history.DataPoints, HistoricalDataPoint{
				Timestamp:    point.Timestamp,
				ResponseTime: time.Duration(point.ResponseTime),
			})
		}

		hs.Services[serviceName] = history
	}

	return nil
}

// SaveToFile saves historical data to JSON file
func (hs *HistoryStore) SaveToFile(filename string) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	rawData := make(map[string][]struct {
		Timestamp    time.Time
		ResponseTime int64
	})

	for serviceName, history := range hs.Services {
		points := make([]struct {
			Timestamp    time.Time
			ResponseTime int64
		}, len(history.DataPoints))

		for i, point := range history.DataPoints {
			points[i].Timestamp = point.Timestamp
			points[i].ResponseTime = int64(point.ResponseTime)
		}

		rawData[serviceName] = points
	}

	data, err := json.MarshalIndent(rawData, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// AddDataPoint adds a new measurement for a service
func (hs *HistoryStore) AddDataPoint(serviceName string, timestamp time.Time, responseTime time.Duration) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.Services[serviceName] == nil {
		hs.Services[serviceName] = &ServiceHistory{
			ServiceName: serviceName,
			DataPoints:  make([]HistoricalDataPoint, 0, 10),
		}
	}

	history := hs.Services[serviceName]
	history.DataPoints = append(history.DataPoints, HistoricalDataPoint{
		Timestamp:    timestamp,
		ResponseTime: responseTime,
	})

	if len(history.DataPoints) > 10 {
		history.DataPoints = history.DataPoints[len(history.DataPoints)-10:]
	}
}

// GetBaseline calculates average of last 10 measurements
func (hs *HistoryStore) GetBaseline(serviceName string) (time.Duration, int) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	history := hs.Services[serviceName]
	if history == nil || len(history.DataPoints) == 0 {
		return 0, 0
	}

	var total time.Duration
	for _, point := range history.DataPoints {
		total += point.ResponseTime
	}

	count := len(history.DataPoints)
	return total / time.Duration(count), count
}

// CalculateTrend determines if current measurement is UP, DOWN, or STEADY
func CalculateTrend(current, baseline time.Duration, sampleCount int) string {
	if sampleCount < 3 {
		return "BASELINE"
	}

	if baseline == 0 {
		return "BASELINE"
	}

	diff := float64(current-baseline) / float64(baseline) * 100

	if diff > 50 {
		return "UP"
	} else if diff < -50 {
		return "DOWN"
	}
	return "STEADY"
}

// resolveDNS resolves hostname to IP and measures time
func resolveDNS(hostname string) (string, time.Duration, error) {
	resolver := &net.Resolver{}

	start := time.Now()
	ips, err := resolver.LookupHost(context.Background(), hostname)
	elapsed := time.Since(start)

	if err != nil {
		return "", 0, err
	}

	if len(ips) == 0 {
		return "", 0, fmt.Errorf("no IPs found")
	}

	return ips[0], elapsed, nil
}

// pingIP pings an IP address (macOS compatible)
func pingIP(ip string) (time.Duration, error) {
	cmd := exec.Command("ping", "-c", "3", "-W", "5000", ip)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ping failed")
	}

	// macOS ping output format: "round-trip min/avg/max/stddev = 10.1/15.2/20.3/5.1 ms"
	re := regexp.MustCompile(`round-trip[^=]+=\s*[\d.]+/([\d.]+)/`)
	matches := re.FindStringSubmatch(string(output))

	if len(matches) > 1 {
		avgMs, _ := strconv.ParseFloat(matches[1], 64)
		return time.Duration(avgMs * float64(time.Millisecond)), nil
	}

	return 0, fmt.Errorf("could not parse ping output")
}

// httpCheck performs HTTP HEAD request
func httpCheck(url string) (time.Duration, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return elapsed, nil
}

// runTest executes a single test
func runTest(endpoint CloudEndpoint, testType TestType, results chan<- TestResult, wg *sync.WaitGroup, history *HistoryStore) {
	defer wg.Done()

	var online bool
	var responseTime time.Duration
	var resolvedIP string
	var errMsg string

	timestamp := time.Now()

	switch testType {
	case TestTypeDNS:
		ip, duration, err := resolveDNS(endpoint.Hostname)
		if err != nil {
			errMsg = err.Error()
		} else {
			online = true
			responseTime = duration
			resolvedIP = ip
		}

	case TestTypePing:
		// First resolve DNS
		ip, _, err := resolveDNS(endpoint.Hostname)
		if err != nil {
			errMsg = "DNS resolution failed"
		} else {
			resolvedIP = ip
			duration, err := pingIP(ip)
			if err != nil {
				errMsg = err.Error()
			} else {
				online = true
				responseTime = duration
			}
		}

	case TestTypeHTTP:
		url := "https://" + endpoint.Hostname
		duration, err := httpCheck(url)
		if err != nil {
			errMsg = err.Error()
		} else {
			online = true
			responseTime = duration
		}
	}

	// Create service key for history
	serviceKey := fmt.Sprintf("%s [%s] - %s", endpoint.Location, endpoint.Provider, testType)

	// Calculate baseline and trend
	baseline, sampleCount := history.GetBaseline(serviceKey)
	trend := CalculateTrend(responseTime, baseline, sampleCount)

	// Add to history if successful
	if online {
		history.AddDataPoint(serviceKey, timestamp, responseTime)
	}

	result := TestResult{
		Endpoint:     endpoint,
		TestType:     testType,
		Online:       online,
		ResponseTime: responseTime,
		ResolvedIP:   resolvedIP,
		Error:        errMsg,
		Timestamp:    timestamp,
		Trend:        trend,
		Baseline:     baseline,
	}

	results <- result
}

// printResult displays a test result
func printResult(result TestResult) {
	var statusColor string
	var status string

	if result.Online {
		statusColor = ColorGreen
		status = "UP"
	} else {
		statusColor = ColorRed
		status = "DOWN"
	}

	// Determine trend color and symbol
	var trendColor string
	var trendSymbol string

	switch result.Trend {
	case "UP":
		trendColor = ColorRed
		trendSymbol = "↑"
	case "DOWN":
		trendColor = ColorGreen
		trendSymbol = "↓"
	case "STEADY":
		trendColor = ColorYellow
		trendSymbol = "→"
	case "BASELINE":
		trendColor = ColorCyan
		trendSymbol = "●"
	}

	locationStr := fmt.Sprintf("%s [%s]", result.Endpoint.Location, result.Endpoint.Provider)

	if result.Online {
		ms := result.ResponseTime.Milliseconds()

		fmt.Printf("%s[%s]%s %-35s %4dms %s[%s%s]%s",
			statusColor, status, ColorReset,
			locationStr, ms,
			trendColor, result.Trend, trendSymbol, ColorReset)

		if result.Trend != "BASELINE" && result.Baseline > 0 {
			baselineMs := result.Baseline.Milliseconds()
			fmt.Printf(" (baseline: %dms)", baselineMs)
		}

		if result.ResolvedIP != "" && result.TestType == TestTypePing {
			fmt.Printf(" [%s]", result.ResolvedIP)
		}

		fmt.Println()
	} else {
		fmt.Printf("%s[%s]%s %-35s %s\n",
			statusColor, status, ColorReset,
			locationStr, result.Error)
	}
}

// writeToLog appends result to log file
func writeToLog(result TestResult, logFile *os.File) {
	if logFile == nil {
		return
	}

	timestamp := result.Timestamp.Format("2006-01-02 15:04:05")
	status := "UP"
	if !result.Online {
		status = "DOWN"
	}

	locationStr := fmt.Sprintf("%s [%s]", result.Endpoint.Location, result.Endpoint.Provider)

	logLine := fmt.Sprintf("%s | [%s] %-35s | Test: %s | Response: %dms | Trend: %s",
		timestamp, status, locationStr,
		result.TestType, result.ResponseTime.Milliseconds(), result.Trend)

	if !result.Online {
		logLine += fmt.Sprintf(" | Error: %s", result.Error)
	}

	logLine += "\n"
	logFile.WriteString(logLine)
}

// runHealthCheck performs one complete health check cycle
func runHealthCheck(endpoints []CloudEndpoint, logFile *os.File, history *HistoryStore) {
	results := make(chan TestResult, len(endpoints)*3)
	var wg sync.WaitGroup

	startTime := time.Now()

	// Launch tests
	for _, endpoint := range endpoints {
		if endpoint.TestDNS {
			wg.Add(1)
			go runTest(endpoint, TestTypeDNS, results, &wg, history)
		}
		if endpoint.TestPing {
			wg.Add(1)
			go runTest(endpoint, TestTypePing, results, &wg, history)
		}
		if endpoint.TestHTTP {
			wg.Add(1)
			go runTest(endpoint, TestTypeHTTP, results, &wg, history)
		}
	}

	wg.Wait()
	close(results)

	// Organize results by test type
	pingResults := []TestResult{}
	dnsResults := []TestResult{}
	httpResults := []TestResult{}

	totalTests := 0
	successfulTests := 0
	var totalResponseTime time.Duration

	for result := range results {
		totalTests++
		if result.Online {
			successfulTests++
			totalResponseTime += result.ResponseTime
		}

		switch result.TestType {
		case TestTypePing:
			pingResults = append(pingResults, result)
		case TestTypeDNS:
			dnsResults = append(dnsResults, result)
		case TestTypeHTTP:
			httpResults = append(httpResults, result)
		}
	}

	// Print results grouped by test type
	if len(pingResults) > 0 {
		fmt.Printf("\n%s=== ICMP PING TESTS (Network Layer Latency) ===%s\n", ColorMagenta, ColorReset)
		for _, result := range pingResults {
			printResult(result)
			writeToLog(result, logFile)
		}
	}

	if len(dnsResults) > 0 {
		fmt.Printf("\n%s=== DNS RESOLUTION TESTS ===%s\n", ColorMagenta, ColorReset)
		for _, result := range dnsResults {
			printResult(result)
			writeToLog(result, logFile)
		}
	}

	if len(httpResults) > 0 {
		fmt.Printf("\n%s=== HTTP/HTTPS TESTS (Application Layer Latency) ===%s\n", ColorMagenta, ColorReset)
		for _, result := range httpResults {
			printResult(result)
			writeToLog(result, logFile)
		}
	}

	// Print summary
	elapsed := time.Since(startTime)
	successRate := float64(successfulTests) / float64(totalTests) * 100
	avgResponseTime := time.Duration(0)
	if successfulTests > 0 {
		avgResponseTime = totalResponseTime / time.Duration(successfulTests)
	}

	fmt.Printf("\n%s=== SUMMARY ===%s", ColorCyan, ColorReset)
	fmt.Printf("\nTotal tests executed: %d", totalTests)
	fmt.Printf("\n%sSuccess rate: %.1f%% (%d/%d)%s",
		ColorGreen, successRate, successfulTests, totalTests, ColorReset)
	fmt.Printf("\nAverage response time: %dms", avgResponseTime.Milliseconds())
	fmt.Printf("\nTotal execution time: %.2fs\n", elapsed.Seconds())

	// Save history
	if err := history.SaveToFile("latency_history.json"); err != nil {
		fmt.Printf("%sWarning: Could not save history: %v%s\n", ColorYellow, err, ColorReset)
	}
}

func main() {
	fmt.Printf("%s=== CLOUD INFRASTRUCTURE LATENCY MONITOR ===%s\n", ColorCyan, ColorReset)
	fmt.Println("Testing AWS regional S3 endpoints")
	fmt.Println("Press Ctrl+C to stop monitoring")

	// Initialize history store
	history := NewHistoryStore()
	if err := history.LoadFromFile("latency_history.json"); err != nil {
		fmt.Printf("%sWarning: Could not load history: %v%s\n", ColorYellow, err, ColorReset)
	} else {
		fmt.Printf("%sLoaded historical data from: latency_history.json%s\n", ColorYellow, ColorReset)
	}

	// Open log file
	logFile, err := os.OpenFile("cloud_latency.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("%sWarning: Could not open log file: %v%s\n",
			ColorYellow, err, ColorReset)
		logFile = nil
	} else {
		defer logFile.Close()
		fmt.Printf("%sLogging to: cloud_latency.log%s\n\n", ColorYellow, ColorReset)
	}

	// Define cloud endpoints to test - All AWS S3 regional endpoints
	endpoints := []CloudEndpoint{
		// Africa
		{Location: "Cape Town, ZA", Region: "af-south-1", Provider: "AWS", Hostname: "s3.af-south-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// South America
		{Location: "São Paulo, BR", Region: "sa-east-1", Provider: "AWS", Hostname: "s3.sa-east-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// Europe
		{Location: "Paris, FR", Region: "eu-west-3", Provider: "AWS", Hostname: "s3.eu-west-3.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Frankfurt, DE", Region: "eu-central-1", Provider: "AWS", Hostname: "s3.eu-central-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "London, UK", Region: "eu-west-2", Provider: "AWS", Hostname: "s3.eu-west-2.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Stockholm, SE", Region: "eu-north-1", Provider: "AWS", Hostname: "s3.eu-north-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Milan, IT", Region: "eu-south-1", Provider: "AWS", Hostname: "s3.eu-south-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// Middle East
		{Location: "Dubai, AE", Region: "me-south-1", Provider: "AWS", Hostname: "s3.me-south-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Riyadh, SA", Region: "me-central-1", Provider: "AWS", Hostname: "s3.me-central-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// Asia - South
		{Location: "Mumbai, IN", Region: "ap-south-1", Provider: "AWS", Hostname: "s3.ap-south-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Hyderabad, IN", Region: "ap-south-2", Provider: "AWS", Hostname: "s3.ap-south-2.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// Asia - Southeast
		{Location: "Singapore, SG", Region: "ap-southeast-1", Provider: "AWS", Hostname: "s3.ap-southeast-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Jakarta, ID", Region: "ap-southeast-3", Provider: "AWS", Hostname: "s3.ap-southeast-3.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// Asia - East
		{Location: "Tokyo, JP", Region: "ap-northeast-1", Provider: "AWS", Hostname: "s3.ap-northeast-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Seoul, KR", Region: "ap-northeast-2", Provider: "AWS", Hostname: "s3.ap-northeast-2.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Osaka, JP", Region: "ap-northeast-3", Provider: "AWS", Hostname: "s3.ap-northeast-3.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// Oceania
		{Location: "Sydney, AU", Region: "ap-southeast-2", Provider: "AWS", Hostname: "s3.ap-southeast-2.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Melbourne, AU", Region: "ap-southeast-4", Provider: "AWS", Hostname: "s3.ap-southeast-4.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// North America - East
		{Location: "Ashburn, VA", Region: "us-east-1", Provider: "AWS", Hostname: "s3.us-east-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Columbus, OH", Region: "us-east-2", Provider: "AWS", Hostname: "s3.us-east-2.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// North America - West
		{Location: "San Jose, CA", Region: "us-west-1", Provider: "AWS", Hostname: "s3.us-west-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
		{Location: "Portland, OR", Region: "us-west-2", Provider: "AWS", Hostname: "s3.us-west-2.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},

		// Canada
		{Location: "Montreal, CA", Region: "ca-central-1", Provider: "AWS", Hostname: "s3.ca-central-1.amazonaws.com", TestPing: true, TestDNS: true, TestHTTP: true},
	}

	interval := 30 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("%s[%s] Starting cloud latency test cycle...%s\n",
		ColorCyan, time.Now().Format("15:04:05"), ColorReset)
	runHealthCheck(endpoints, logFile, history)

	for range ticker.C {
		fmt.Printf("\n%s[%s] Starting cloud latency test cycle...%s\n",
			ColorCyan, time.Now().Format("15:04:05"), ColorReset)
		runHealthCheck(endpoints, logFile, history)
	}
}
