# Cloud Infrastructure Latency Monitor

A professional-grade, real-time latency monitoring system that tests AWS global infrastructure across 23 regions with comprehensive historical tracking, trend analysis, and baseline performance detection.

## Features

- ✅ **Multi-layer latency testing** - ICMP ping, DNS resolution, and HTTP/HTTPS checks
- ✅ **23 AWS regions worldwide** - True infrastructure endpoints (no CDN interference)
- ✅ **Real-time trend detection** - Automatically detects performance degradation (↑), improvement (↓), or stability (→)
- ✅ **Historical baseline tracking** - Maintains last 10 measurements per endpoint to establish performance baselines
- ✅ **Persistent data storage** - All measurements saved to JSON with timestamps for day-over-day analysis
- ✅ **Grouped test results** - Organized by test type (Ping, DNS, HTTP) for easy comparison
- ✅ **Concurrent execution** - All tests run simultaneously using goroutines
- ✅ **Color-coded output** - Visual indicators for status and trends
- ✅ **Comprehensive logging** - All results logged to file for historical analysis

## What This Measures

### ICMP Ping Tests (Network Layer)
Pure network latency with no application overhead. Shows the actual time for packets to travel from your location to AWS data centers worldwide.

### DNS Resolution Tests
Measures how quickly AWS regional hostnames can be resolved to IP addresses. Important for understanding DNS infrastructure performance.

### HTTP/HTTPS Tests (Application Layer)
Complete application-level latency including TLS handshake, connection establishment, and HTTP protocol overhead. Most representative of real-world application performance.

## Global Coverage

Testing AWS S3 endpoints across:
- **North America**: Virginia, Ohio, California, Oregon, Montreal
- **South America**: São Paulo
- **Europe**: London, Paris, Frankfurt, Stockholm, Milan
- **Middle East**: Dubai, Riyadh
- **Asia**: Mumbai, Hyderabad, Singapore, Jakarta, Tokyo, Seoul, Osaka
- **Africa**: Cape Town
- **Oceania**: Sydney, Melbourne

## Prerequisites

- Go 1.23 or higher
- macOS, Linux, or Windows
- Network connectivity
- Terminal with ANSI color support

## Installation
```bash
git clone https://github.com/MaddaloniG3/home-health-monitor.git
cd home-health-monitor
go run main.go
```

## Usage

The monitor runs continuously, testing all endpoints every 30 seconds:
```bash
go run main.go
```

**To stop monitoring:** Press `Ctrl+C`

### Example Output
```
=== CLOUD INFRASTRUCTURE LATENCY MONITOR ===
Testing AWS regional S3 endpoints
Press Ctrl+C to stop monitoring

Loaded historical data from: latency_history.json
Logging to: cloud_latency.log

[21:56:01] Starting cloud latency test cycle...

=== ICMP PING TESTS (Network Layer Latency) ===
[UP] Ashburn, VA [AWS]                     16ms [BASELINE●] [16.15.178.220]
[UP] Montreal, CA [AWS]                    18ms [BASELINE●] [3.5.254.81]
[UP] Columbus, OH [AWS]                    29ms [BASELINE●] [3.5.130.1]
[UP] London, UK [AWS]                      80ms [BASELINE●] [3.5.245.32]
[UP] Singapore, SG [AWS]                  244ms [STEADY→] (baseline: 240ms)

=== DNS RESOLUTION TESTS ===
[UP] Singapore, SG [AWS]                   41ms [DOWN↓] (baseline: 52ms)
[UP] Frankfurt, DE [AWS]                   51ms [STEADY→] (baseline: 49ms)
[UP] Tokyo, JP [AWS]                       55ms [UP↑] (baseline: 32ms)

=== HTTP/HTTPS TESTS (Application Layer Latency) ===
[UP] Ashburn, VA [AWS]                    137ms [STEADY→] (baseline: 135ms)
[UP] Cape Town, ZA [AWS]                  839ms [DOWN↓] (baseline: 1200ms)

=== SUMMARY ===
Total tests executed: 69
Success rate: 98.6% (68/69)
Average response time: 245ms
Total execution time: 4.10s
```

## Understanding Trends

After collecting 3+ samples, the system shows performance trends:

- 🔴 **UP ↑** (Red) - Latency increased >50% vs baseline (degradation)
- 🟢 **DOWN ↓** (Green) - Latency decreased >50% vs baseline (improvement)
- 🟡 **STEADY →** (Yellow) - Latency within ±50% of baseline (normal)
- 🔵 **BASELINE ●** (Cyan) - Still building baseline (<3 samples)

## Data Files

### latency_history.json
Stores the last 10 measurements for each endpoint with timestamps. Used for:
- Calculating rolling baselines
- Trend detection
- Day-over-day comparisons
- Historical analysis

### cloud_latency.log
Complete log of all tests with timestamps, status, response times, and trends. Format:
```
2024-12-28 21:56:01 | [UP] Ashburn, VA [AWS] | Test: PING | Response: 16ms | Trend: BASELINE
2024-12-28 21:56:01 | [UP] Singapore, SG [AWS] | Test: HTTP | Response: 792ms | Trend: STEADY
```

## Performance Analysis

Run the analysis tool to generate statistical summaries:
```bash
go run analyze_history.go
```

This produces a comprehensive table showing:
- Min/Max/Average latency per endpoint
- Number of measurements collected
- Performance trends (first vs last measurement)
- Fastest and slowest services
- Most improved and degraded endpoints

## Technical Architecture

### Concurrent Testing
Uses Go's goroutines to run all tests simultaneously:
- 23 regions × 3 test types = 69 concurrent tests
- Completes in ~4 seconds despite checking 69 endpoints
- Uses channels for safe result communication
- WaitGroups ensure all tests complete before reporting

### Historical Tracking
- Maintains sliding window of last 10 measurements per endpoint
- Calculates rolling average as baseline
- Compares current measurement to baseline for trend detection
- Persists to JSON after each cycle

### Test Methodology

**ICMP Ping:**
1. Resolve hostname to IP via DNS
2. Send 3 ICMP packets to resolved IP
3. Parse average round-trip time from system ping output
4. Reports actual IP address tested

**DNS Resolution:**
1. Measure time to resolve hostname to IP
2. Uses system resolver
3. Returns first IP from results

**HTTP/HTTPS:**
1. Send HTTP HEAD request (lightweight, no body transfer)
2. Includes full TLS handshake for HTTPS
3. Measures total time to receive headers
4. Uses 10-second timeout

### Why AWS S3 Endpoints?

AWS S3 regional endpoints are ideal for latency testing because:
- **Guaranteed regional placement** - No CDN or edge caching
- **Highly available** - 99.99% uptime SLA
- **Globally distributed** - 23 regions across 6 continents
- **Consistent infrastructure** - Standardized endpoints across regions
- **Free to test** - HEAD requests don't incur charges

## Customization

### Change Monitoring Interval

Edit the interval in `main.go`:
```go
interval := 30 * time.Second  // Default: 30 seconds
```

Options:
- `10 * time.Second` - Every 10 seconds (more frequent)
- `1 * time.Minute` - Every minute
- `5 * time.Minute` - Every 5 minutes

### Add/Remove Regions

Edit the `endpoints` slice in `main.go`:
```go
endpoints := []CloudEndpoint{
    {Location: "Your City", Region: "aws-region", Provider: "AWS", 
     Hostname: "s3.aws-region.amazonaws.com", 
     TestPing: true, TestDNS: true, TestHTTP: true},
}
```

### Disable Test Types

Set flags to `false` to skip specific test types:
```go
{Location: "Tokyo, JP", Region: "ap-northeast-1", Provider: "AWS",
 Hostname: "s3.ap-northeast-1.amazonaws.com",
 TestPing: false,  // Skip ping tests
 TestDNS: true,
 TestHTTP: true},
```

## Use Cases

- **Global infrastructure monitoring** - Track AWS availability from your location
- **Performance baselining** - Establish normal latency patterns
- **Trend detection** - Identify degrading network paths early
- **Region selection** - Choose optimal AWS regions for deployments
- **Network troubleshooting** - Isolate whether issues are DNS, network, or application layer
- **Day/night comparison** - See if latency varies by time of day
- **ISP performance** - Monitor your internet provider's routing efficiency

## Real-World Insights

From testing in White Plains, NY:

**Fastest Regions (Ping):**
- Ashburn, VA: 16ms (closest AWS region)
- Montreal, CA: 18ms
- Columbus, OH: 29ms

**Expected Latencies:**
- North America: 15-80ms
- Europe: 80-110ms
- South America: 125ms
- Asia: 180-260ms
- Africa: 240ms

**Application Layer Overhead:**
- Ping to HTTP typically adds 100-150ms (TLS handshake + HTTP protocol)
- DNS resolution: 40-110ms depending on caching

## Built With

- **Language**: Go 1.23+
- **Standard Library Only**: No external dependencies
- **System Integration**: Uses native `ping` command for ICMP tests
- **Storage**: JSON for data persistence
- **Output**: ANSI colors for terminal display

## Core Technologies

- `net/http` - HTTP client and HEAD requests
- `net` - DNS resolution
- `os/exec` - System ping command execution
- `regexp` - Parsing ping output
- `encoding/json` - Data persistence
- `sync` - Goroutines, channels, WaitGroups
- `time` - Timestamps and scheduling
- `crypto/tls` - HTTPS support

## Learning Journey

This project demonstrates advanced Go concepts:
- **Concurrency**: 69 simultaneous tests using goroutines
- **Channels**: Safe communication between concurrent tests
- **Historical data**: Time-series tracking with trend analysis
- **File I/O**: JSON serialization and persistence
- **System integration**: Executing and parsing system commands
- **Data structures**: Nested maps for multi-dimensional data
- **Statistical analysis**: Baseline calculation and variance detection

## Author

**George Maddaloni**  
CTO of Operations, Mastercard Networks  
[www.georgemaddaloni.com](https://www.georgemaddaloni.com)

Professional expertise in global network infrastructure, payment processing, and enterprise technology platforms.

## License

This project is open source and available for personal and educational use.

## Acknowledgments

Built as part of the "Learning GO" project to develop production-ready monitoring capabilities while mastering Go's concurrency model and system integration features.

---

## Quick Start
```bash
# Clone repository
git clone https://github.com/MaddaloniG3/home-health-monitor.git
cd home-health-monitor

# Run monitor
go run main.go

# In another terminal, analyze results (after collecting data)
go run analyze_history.go
```

---

**Project Status:** Production Ready ✅  
**Current Version:** 2.0 - Cloud Infrastructure Monitor  
**Last Updated:** December 28, 2024  
**Total AWS Regions Tested:** 23 across 6 continents