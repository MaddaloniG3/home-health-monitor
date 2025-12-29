# Home Network Health Monitor

A lightweight, concurrent network monitoring tool written in Go that checks the health and connectivity of network services and endpoints with real-time response time measurements.

## Features

- ✅ **Concurrent health checks** - Check multiple services simultaneously using goroutines
- ✅ **Response time measurement** - Track individual response times for each service
- ✅ **HTTP/HTTPS endpoint monitoring** - Support for both protocols
- ✅ **Self-signed certificate support** - Monitor local devices with custom certificates
- ✅ **Clean, formatted output** - Easy-to-read status reports with aligned columns
- ✅ **Fast execution** - Parallel checks complete in seconds, not minutes

## Prerequisites

- Go 1.23 or higher
- Network connectivity

## Installation

1. Clone this repository:
```bash
git clone https://github.com/MaddaloniG3/home-health-monitor.git
cd home-health-monitor
```

2. Run the monitor:
```bash
go run main.go
```

## Usage

The monitor checks a predefined list of services concurrently and reports their status with response times:
```bash
go run main.go
```

Example output:
```
=== Network Health Monitor ===

Checking Home Router at https://192.168.3.1...
Checking Google at https://www.google.com...
Checking GitHub at https://github.com...
Checking Cape Town, SA (UCT) at https://www.uct.ac.za...

=== Results ===
[UP] Home Router                    Response time: 0.04s
[UP] GitHub                         Response time: 0.11s
[UP] Google                         Response time: 0.19s
[UP] Cape Town, SA (UCT)            Response time: 0.97s

=== Summary ===
Total services checked: 4
Total execution time: 0.97s
```

## Configuration

To monitor your own services, edit the `services` slice in `main.go`:
```go
services := []Service{
    {Name: "My Router", URL: "https://192.168.1.1", Insecure: true},
    {Name: "My Server", URL: "https://myserver.local", Insecure: false},
    {Name: "External API", URL: "https://api.example.com", Insecure: false},
}
```

**Service struct fields:**
- `Name`: Display name for the service
- `URL`: Full URL to check (http:// or https://)
- `Insecure`: Set to `true` for local devices with self-signed certificates

## How It Works

### Concurrent Execution
The monitor uses Go's goroutines to check all services simultaneously rather than sequentially. This means:
- 4 services complete in ~1 second instead of 4+ seconds
- Scales efficiently - 20 services still complete in seconds
- Uses Go's channels for safe communication between goroutines
- WaitGroups ensure all checks complete before reporting results

### Response Time Measurement
Each service check is timed individually:
- Measures actual HTTP request/response time
- Formatted to 2 decimal places for readability
- Total execution time shows the longest individual check (due to concurrency)

## Technical Implementation

**Key Go concepts demonstrated:**
- Goroutines for concurrent execution
- Channels for safe data passing
- WaitGroups for synchronization
- Structs for data modeling
- Slices for dynamic collections
- HTTP client with custom TLS configuration
- Time measurement and formatting

## Roadmap

- [x] Concurrent health checks for faster execution
- [x] Individual response time measurements
- [ ] Periodic monitoring with configurable intervals
- [ ] Logging results to file with timestamps
- [ ] ICMP ping support for non-HTTP devices
- [ ] Email/Slack notifications for downtime
- [ ] Web dashboard for visualization
- [ ] Historical data and trend analysis

## Performance

Typical execution times (MacBook M2):
- Local network devices: 40-100ms
- US-based services: 100-200ms
- International services: 900ms-2s
- Total runtime: ~1s (limited by slowest service, not sum of all services)

## Built With

- [Go](https://golang.org/) - The Go Programming Language
- Standard library packages: `net/http`, `crypto/tls`, `time`, `sync`

## Author

George Maddaloni - CTO of Operations at Mastercard Networks

## License

This project is open source and available for personal and educational use.

## Learning Journey

This project was created as part of learning Go programming, with a focus on:
- Go basics (variables, functions, structs, slices)
- Working with data structures
- HTTP client operations and TLS/SSL certificate handling
- **Concurrency patterns** (goroutines, channels, WaitGroups)
- **Performance optimization** through parallel execution
- Error handling patterns in Go
- Code organization and modularity

---

### Project Status: Phase 2 Complete ✅

**Phase 1:** Basic health checks with sequential execution  
**Phase 2:** Concurrent execution with response time measurements ✅  
**Phase 3:** Periodic monitoring (in progress)