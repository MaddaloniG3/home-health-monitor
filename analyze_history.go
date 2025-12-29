package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "sort"
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
    
    // Calculate statistics for each service
    type ServiceStats struct {
        Name         string
        Count        int
        MinMs        int64
        MaxMs        int64
        AvgMs        int64
        LastMs       int64
        FirstTime    time.Time
        LastTime     time.Time
        TrendPercent float64
    }
    
    var stats []ServiceStats
    
    for serviceName, dataPoints := range history {
        if len(dataPoints) == 0 {
            continue
        }
        
        var totalMs int64
        minMs := int64(999999999)
        maxMs := int64(0)
        
        for _, point := range dataPoints {
            ms := point.ResponseTime / 1000000 // Convert nanoseconds to milliseconds
            totalMs += ms
            
            if ms < minMs {
                minMs = ms
            }
            if ms > maxMs {
                maxMs = ms
            }
        }
        
        avgMs := totalMs / int64(len(dataPoints))
        lastMs := dataPoints[len(dataPoints)-1].ResponseTime / 1000000
        firstMs := dataPoints[0].ResponseTime / 1000000
        
        // Calculate trend (comparing last measurement to first)
        trendPercent := 0.0
        if firstMs > 0 {
            trendPercent = float64(lastMs-firstMs) / float64(firstMs) * 100
        }
        
        stats = append(stats, ServiceStats{
            Name:         serviceName,
            Count:        len(dataPoints),
            MinMs:        minMs,
            MaxMs:        maxMs,
            AvgMs:        avgMs,
            LastMs:       lastMs,
            FirstTime:    dataPoints[0].Timestamp,
            LastTime:     dataPoints[len(dataPoints)-1].Timestamp,
            TrendPercent: trendPercent,
        })
    }
    
    // Sort by name
    sort.Slice(stats, func(i, j int) bool {
        return stats[i].Name < stats[j].Name
    })
    
    // Print header
    fmt.Println("\n╔════════════════════════════════════════════════════════════════════════════════════════╗")
    fmt.Println("║                     CLOUD LATENCY HISTORY ANALYSIS                                     ║")
    fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════╝")
    fmt.Printf("\nAnalysis run: %s\n", time.Now().Format("2006-01-02 15:04:05"))
    fmt.Printf("Total endpoints tracked: %d\n\n", len(stats))
    
    // Print table header
    fmt.Printf("%-60s %6s %8s %8s %8s %8s %10s\n", 
        "ENDPOINT", "COUNT", "MIN(ms)", "AVG(ms)", "MAX(ms)", "LAST(ms)", "TREND")
    fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────")
    
    // Print each service
    for _, s := range stats {
        // Determine trend symbol
        trendSymbol := "→"
        if s.TrendPercent > 50 {
            trendSymbol = "↑"
        } else if s.TrendPercent < -50 {
            trendSymbol = "↓"
        }
        
        fmt.Printf("%-60s %6d %8d %8d %8d %8d %7.1f%% %s\n",
            s.Name, s.Count, s.MinMs, s.AvgMs, s.MaxMs, s.LastMs, s.TrendPercent, trendSymbol)
    }
    
    fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────")
    
    // Print summary by test type
    fmt.Println("\n╔════════════════════════════════════════════════════════════════════════════════════════╗")
    fmt.Println("║                              SUMMARY STATISTICS                                        ║")
    fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════╝\n")
    
    // Find fastest and slowest
    var fastest, slowest ServiceStats
    fastest.AvgMs = 999999
    
    for _, s := range stats {
        if s.AvgMs < fastest.AvgMs {
            fastest = s
        }
        if s.AvgMs > slowest.AvgMs {
            slowest = s
        }
    }
    
    fmt.Printf("Fastest endpoint:  %-60s %d ms average\n", fastest.Name, fastest.AvgMs)
    fmt.Printf("Slowest endpoint:  %-60s %d ms average\n", slowest.Name, slowest.AvgMs)
    
    // Find most improved and most degraded
    var mostImproved, mostDegraded ServiceStats
    mostImproved.TrendPercent = 999999
    mostDegraded.TrendPercent = -999999
    
    for _, s := range stats {
        if s.TrendPercent < mostImproved.TrendPercent {
            mostImproved = s
        }
        if s.TrendPercent > mostDegraded.TrendPercent {
            mostDegraded = s
        }
    }
    
    fmt.Printf("\nMost improved:     %-60s %.1f%% faster\n", mostImproved.Name, -mostImproved.TrendPercent)
    fmt.Printf("Most degraded:     %-60s %.1f%% slower\n", mostDegraded.Name, mostDegraded.TrendPercent)
    
    // Time range
    if len(stats) > 0 {
        fmt.Printf("\nData collected from: %s to %s\n", 
            stats[0].FirstTime.Format("2006-01-02 15:04:05"),
            stats[0].LastTime.Format("2006-01-02 15:04:05"))
        
        duration := stats[0].LastTime.Sub(stats[0].FirstTime)
        fmt.Printf("Collection period: %v\n", duration.Round(time.Second))
    }
    
    fmt.Println()
}