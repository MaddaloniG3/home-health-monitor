package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"
)

// DataPoint represents a single measurement from the JSON file
type DataPoint struct {
	Timestamp    time.Time
	ResponseTime int64 // nanoseconds
}

func main() {
	// Read the JSON file
	data, err := ioutil.ReadFile("latency_history.json")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON
	var history map[string][]DataPoint
	if err := json.Unmarshal(data, &history); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generating CSV exports...")

	// Export 1: Summary Statistics
	if err := exportSummary(history); err != nil {
		fmt.Printf("Error exporting summary: %v\n", err)
	} else {
		fmt.Println("✓ Created: latency_summary.csv")
	}

	// Export 2: Time Series (all measurements)
	if err := exportTimeSeries(history); err != nil {
		fmt.Printf("Error exporting time series: %v\n", err)
	} else {
		fmt.Println("✓ Created: latency_timeseries.csv")
	}

	// Export 3: Latest Measurements
	if err := exportLatest(history); err != nil {
		fmt.Printf("Error exporting latest: %v\n", err)
	} else {
		fmt.Println("✓ Created: latency_latest.csv")
	}

	// Export 4: Pivot by Test Type
	if err := exportByTestType(history); err != nil {
		fmt.Printf("Error exporting by test type: %v\n", err)
	} else {
		fmt.Println("✓ Created: latency_by_test_type.csv")
	}

	fmt.Println("\nAll CSV files generated successfully!")
	fmt.Println("Open in Excel for analysis and visualization.")
}

// exportSummary creates a summary statistics CSV
func exportSummary(history map[string][]DataPoint) error {
	file, err := os.Create("latency_summary.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Endpoint", "Test Type", "Location", "Provider", "Sample Count",
		"Min (ms)", "Average (ms)", "Max (ms)", "Std Dev (ms)",
		"Latest (ms)", "First (ms)", "Trend (%)", "Status"}
	writer.Write(header)

	// Collect and sort endpoints
	type EndpointStats struct {
		Name     string
		TestType string
		Location string
		Provider string
		Count    int
		MinMs    int64
		AvgMs    int64
		MaxMs    int64
		StdDevMs float64
		LatestMs int64
		FirstMs  int64
		TrendPct float64
		Status   string
	}

	var stats []EndpointStats

	for serviceName, dataPoints := range history {
		if len(dataPoints) == 0 {
			continue
		}

		// Parse service name to extract components
		// Format: "Location [Provider] - TestType"
		location, provider, testType := parseServiceName(serviceName)

		var totalMs int64
		var values []int64
		minMs := int64(999999999)
		maxMs := int64(0)

		for _, point := range dataPoints {
			ms := point.ResponseTime / 1000000
			values = append(values, ms)
			totalMs += ms

			if ms < minMs {
				minMs = ms
			}
			if ms > maxMs {
				maxMs = ms
			}
		}

		avgMs := totalMs / int64(len(dataPoints))
		latestMs := dataPoints[len(dataPoints)-1].ResponseTime / 1000000
		firstMs := dataPoints[0].ResponseTime / 1000000

		// Calculate standard deviation
		var variance float64
		for _, val := range values {
			diff := float64(val - avgMs)
			variance += diff * diff
		}
		variance /= float64(len(values))
		stdDev := 0.0
		if variance > 0 {
			stdDev = float64(int64(1000*(variance*variance))) / 1000 // Simple sqrt approximation
		}

		// Calculate trend
		trendPct := 0.0
		if firstMs > 0 {
			trendPct = float64(latestMs-firstMs) / float64(firstMs) * 100
		}

		// Determine status
		status := "STEADY"
		if trendPct > 50 {
			status = "DEGRADED"
		} else if trendPct < -50 {
			status = "IMPROVED"
		}

		stats = append(stats, EndpointStats{
			Name:     serviceName,
			TestType: testType,
			Location: location,
			Provider: provider,
			Count:    len(dataPoints),
			MinMs:    minMs,
			AvgMs:    avgMs,
			MaxMs:    maxMs,
			StdDevMs: stdDev,
			LatestMs: latestMs,
			FirstMs:  firstMs,
			TrendPct: trendPct,
			Status:   status,
		})
	}

	// Sort by location then test type
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Location == stats[j].Location {
			return stats[i].TestType < stats[j].TestType
		}
		return stats[i].Location < stats[j].Location
	})

	// Write data
	for _, s := range stats {
		row := []string{
			s.Name,
			s.TestType,
			s.Location,
			s.Provider,
			strconv.Itoa(s.Count),
			strconv.FormatInt(s.MinMs, 10),
			strconv.FormatInt(s.AvgMs, 10),
			strconv.FormatInt(s.MaxMs, 10),
			fmt.Sprintf("%.2f", s.StdDevMs),
			strconv.FormatInt(s.LatestMs, 10),
			strconv.FormatInt(s.FirstMs, 10),
			fmt.Sprintf("%.2f", s.TrendPct),
			s.Status,
		}
		writer.Write(row)
	}

	return nil
}

// exportTimeSeries creates a time-series CSV with all measurements
func exportTimeSeries(history map[string][]DataPoint) error {
	file, err := os.Create("latency_timeseries.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Timestamp", "Endpoint", "Test Type", "Location", "Provider", "Response Time (ms)"}
	writer.Write(header)

	// Collect all measurements
	type Measurement struct {
		Timestamp time.Time
		Endpoint  string
		TestType  string
		Location  string
		Provider  string
		ValueMs   int64
	}

	var measurements []Measurement

	for serviceName, dataPoints := range history {
		location, provider, testType := parseServiceName(serviceName)

		for _, point := range dataPoints {
			measurements = append(measurements, Measurement{
				Timestamp: point.Timestamp,
				Endpoint:  serviceName,
				TestType:  testType,
				Location:  location,
				Provider:  provider,
				ValueMs:   point.ResponseTime / 1000000,
			})
		}
	}

	// Sort by timestamp
	sort.Slice(measurements, func(i, j int) bool {
		return measurements[i].Timestamp.Before(measurements[j].Timestamp)
	})

	// Write data
	for _, m := range measurements {
		row := []string{
			m.Timestamp.Format("2006-01-02 15:04:05"),
			m.Endpoint,
			m.TestType,
			m.Location,
			m.Provider,
			strconv.FormatInt(m.ValueMs, 10),
		}
		writer.Write(row)
	}

	return nil
}

// exportLatest creates CSV with just the latest measurement for each endpoint
func exportLatest(history map[string][]DataPoint) error {
	file, err := os.Create("latency_latest.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Endpoint", "Test Type", "Location", "Provider",
		"Latest Response Time (ms)", "Timestamp", "vs Baseline (%)", "Status"}
	writer.Write(header)

	type LatestMeasurement struct {
		Endpoint   string
		TestType   string
		Location   string
		Provider   string
		LatestMs   int64
		Timestamp  time.Time
		BaselineMs int64
		TrendPct   float64
		Status     string
	}

	var latest []LatestMeasurement

	for serviceName, dataPoints := range history {
		if len(dataPoints) == 0 {
			continue
		}

		location, provider, testType := parseServiceName(serviceName)

		// Get latest measurement
		lastPoint := dataPoints[len(dataPoints)-1]
		latestMs := lastPoint.ResponseTime / 1000000

		// Calculate baseline (average of all measurements)
		var totalMs int64
		for _, point := range dataPoints {
			totalMs += point.ResponseTime / 1000000
		}
		baselineMs := totalMs / int64(len(dataPoints))

		// Calculate trend vs baseline
		trendPct := 0.0
		if baselineMs > 0 {
			trendPct = float64(latestMs-baselineMs) / float64(baselineMs) * 100
		}

		status := "NORMAL"
		if trendPct > 50 {
			status = "SLOW"
		} else if trendPct < -50 {
			status = "FAST"
		}

		latest = append(latest, LatestMeasurement{
			Endpoint:   serviceName,
			TestType:   testType,
			Location:   location,
			Provider:   provider,
			LatestMs:   latestMs,
			Timestamp:  lastPoint.Timestamp,
			BaselineMs: baselineMs,
			TrendPct:   trendPct,
			Status:     status,
		})
	}

	// Sort by location
	sort.Slice(latest, func(i, j int) bool {
		if latest[i].Location == latest[j].Location {
			return latest[i].TestType < latest[j].TestType
		}
		return latest[i].Location < latest[j].Location
	})

	// Write data
	for _, l := range latest {
		row := []string{
			l.Endpoint,
			l.TestType,
			l.Location,
			l.Provider,
			strconv.FormatInt(l.LatestMs, 10),
			l.Timestamp.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.2f", l.TrendPct),
			l.Status,
		}
		writer.Write(row)
	}

	return nil
}

// exportByTestType creates separate columns for each test type
func exportByTestType(history map[string][]DataPoint) error {
	file, err := os.Create("latency_by_test_type.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Organize data by location and test type
	type LocationData struct {
		Location string
		Provider string
		PingMs   int64
		DNSMs    int64
		HTTPMs   int64
	}

	locationMap := make(map[string]*LocationData)

	for serviceName, dataPoints := range history {
		if len(dataPoints) == 0 {
			continue
		}

		location, provider, testType := parseServiceName(serviceName)

		// Get average latency
		var totalMs int64
		for _, point := range dataPoints {
			totalMs += point.ResponseTime / 1000000
		}
		avgMs := totalMs / int64(len(dataPoints))

		// Create or get location entry
		key := location + "-" + provider
		if locationMap[key] == nil {
			locationMap[key] = &LocationData{
				Location: location,
				Provider: provider,
			}
		}

		// Store by test type
		switch testType {
		case "PING":
			locationMap[key].PingMs = avgMs
		case "DNS":
			locationMap[key].DNSMs = avgMs
		case "HTTP":
			locationMap[key].HTTPMs = avgMs
		}
	}

	// Convert to slice and sort
	var locations []LocationData
	for _, data := range locationMap {
		locations = append(locations, *data)
	}

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Location < locations[j].Location
	})

	// Write header
	header := []string{"Location", "Provider", "PING (ms)", "DNS (ms)", "HTTP (ms)", "Total Latency (ms)"}
	writer.Write(header)

	// Write data
	for _, loc := range locations {
		total := loc.PingMs + loc.DNSMs + loc.HTTPMs
		row := []string{
			loc.Location,
			loc.Provider,
			strconv.FormatInt(loc.PingMs, 10),
			strconv.FormatInt(loc.DNSMs, 10),
			strconv.FormatInt(loc.HTTPMs, 10),
			strconv.FormatInt(total, 10),
		}
		writer.Write(row)
	}

	return nil
}

// parseServiceName extracts location, provider, and test type from service name
func parseServiceName(name string) (location, provider, testType string) {
	// Format examples:
	// "Ashburn, VA [AWS] - PING"
	// "Home Router"
	// "GitHub"

	testType = "OTHER"
	provider = "N/A"
	location = name

	// Extract test type (after last " - ")
	if idx := len(name) - 1; idx > 0 {
		for i := len(name) - 1; i >= 0; i-- {
			if i >= 3 && name[i-2:i+1] == " - " {
				testType = name[i+1:]
				name = name[:i-2]
				break
			}
		}
	}

	// Extract provider (between [ and ])
	start := -1
	end := -1
	for i, ch := range name {
		if ch == '[' {
			start = i
		} else if ch == ']' {
			end = i
			break
		}
	}

	if start >= 0 && end > start {
		provider = name[start+1 : end]
		location = name[:start-1] // Remove space before bracket
	}

	return location, provider, testType
}
