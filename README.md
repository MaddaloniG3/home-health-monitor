# Home Network Health Monitor

A professional-grade, concurrent network monitoring tool written in Go that provides real-time health checks and connectivity monitoring for network services and endpoints with comprehensive logging and statistics.

## Features

- ✅ **Concurrent health checks** - Check multiple services simultaneously using goroutines
- ✅ **Dual monitoring modes** - HTTP/HTTPS endpoint monitoring and ICMP ping checks
- ✅ **Real-time response time measurement** - Track individual response times for each service
- ✅ **Periodic monitoring** - Automated checks every 30 seconds with continuous monitoring
- ✅ **Color-coded output** - Visual status indicators (green for UP, red for DOWN)
- ✅ **Timestamped results** - Each check includes precise timestamp for tracking
- ✅ **Persistent logging** - Automatic logging to file for historical analysis
- ✅ **Success rate statistics** - Real-time calculation of uptime percentage
- ✅ **Average response time tracking** - Performance metrics across all services
- ✅ **Self-signed certificate support** - Monitor local devices with custom certificates
- ✅ **Fast execution** - Parallel checks complete in ~2 seconds regardless of service count

## Prerequisites

- Go 1.23 or higher
- macOS, Linux, or Windows
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

The monitor runs continuously, checking all configured services every 30 seconds:
```bash
go run main.go
```

**To stop monitoring:** Press `Ctrl+C`

### Example Output
```
=== Network Health Monitor ===
Press Ctrl+C to stop monitoring

Logging to: health_monitor.log

[20:53:20] Starting health check cycle...

=== Results ===
[UP] [20:53:20] Home Router                    [http] Response time: 0.03s
[UP] [20:53:20] GitHub                         [http] Response time: 0.08s
[UP] [20:53:20] Mastercard Website             [http] Response time: 0.12s
[UP] [20:53:21] Google                         [http] Response time: 0.18s
[UP] [20:53:21] George Maddaloni Website       [http] Response time: 0.24s
[UP] [20:53:21] MA Connect Website             [http] Response time: 0.34s
[UP] [20:53:22] Cape Town, SA (UCT)            [http] Response time: 1.26s
[UP] [20:53:22] Router (ping)                  [ping] Response time: 0.68s
[UP] [20:53:22] Google DNS                     [ping] Response time: 0.68s

=== Summary ===
Total services checked: 9
Success rate: 100.0% (9/9)
Average response time: 0.40s
Total execution time: 2.03s
```

## Configuration

Edit the `services` slice in `main.go` to customize your monitoring targets:
```go
services := []Service{
    // HTTP/HTTPS checks
    {Name: "My Website", URL: "https://example.com", Type: TypeHTTP, Insecure: false},
    {Name: "My Router", URL: "https://192.168.1.1", Type: TypeHTTP, Insecure: true},
    
    // Ping checks
    {Name: "My Server", Host: "192.168.1.100", Type: TypePing},
    {Name: "Google DNS", Host: "8.8.8.8", Type: TypePing},
}
```

### Service Configuration Fields

**For HTTP/HTTPS checks:**
- `Name`: Display name for the service
- `URL`: Full URL to check (must include http:// or https://)
- `Type`: Set to `TypeHTTP`
- `Insecure`: Set to `true` for local devices with self-signed certificates, `false` otherwise

**For Ping checks:**
- `Name`: Display name for the service
- `Host`: IP address or hostname to ping
- `Type`: Set to `TypePing`

### Customizing Check Interval

Change the monitoring interval by modifying this line in `main.go`:
```go
interval := 30 * time.Second  // Change to desired interval
```

Examples:
- `10 * time.Second` - Check every 10 seconds
- `1 * time.Minute` - Check every minute
- `5 * time.Minute` - Check every 5 minutes

## Output Features

### Color Coding
- 🟢 **Green [UP]** - Service is responding normally
- 🔴 **Red [DOWN]** - Service is unreachable or failing
- 🔵 **Cyan** - Section headers and timestamps
- 🟡 **Yellow** - Warnings and informational messages

### Logging
All checks are automatically logged to `health_monitor.log` with the format:
```
2024-12-28 20:53:20 | [UP] Home Router | Type: http | Response: 0.03s
2024-12-28 20:53:20 | [DOWN] Test Service | Type: http | Response: 0.00s | Error: timeout
```

### Statistics
Each monitoring cycle displays:
- **Total services checked** - Number of configured services
- **Success rate** - Percentage of services responding successfully
- **Average response time** - Mean response time across all successful checks
- **Total execution time** - Time to complete all checks (typically ~2s due to concurrency)

## How It Works

### Concurrent Execution Architecture
The monitor leverages Go's powerful concurrency primitives:

- **Goroutines** - Each service check runs in a separate lightweight thread
- **Channels** - Safe communication between concurrent checks
- **WaitGroups** - Synchronization to ensure all checks complete before reporting
- **Result** - 9 services complete in ~2 seconds instead of 9+ seconds sequentially

### Performance Characteristics
- **Scalability**: Adding more services has minimal impact on total execution time
- **Efficiency**: Checking 20 services takes roughly the same time as checking 5
- **Limiting factor**: Total time is determined by the slowest individual check, not the sum

### Ping Implementation
Uses the macOS/Linux system `ping` command for ICMP checks:
- Sends 3 packets per check
- 5-second timeout
- Parses average round-trip time from output
- Falls back to TCP connection test if ping fails

## Technical Implementation

### Core Technologies
- **Language**: Go 1.23+
- **Standard Library Packages**: 
  - `net/http` - HTTP client operations
  - `crypto/tls` - TLS/SSL certificate handling
  - `sync` - Concurrency primitives (WaitGroups, channels)
  - `time` - Time measurement and scheduling
  - `os/exec` - System command execution for ping
  - `regexp` - Parsing ping output
  - `os` - File operations for logging

### Go Concepts Demonstrated
- Structs and custom types for data modeling
- Slices for dynamic collections
- Goroutines for concurrent execution
- Channels for inter-goroutine communication
- WaitGroups for synchronization
- Switch statements for type handling
- Error handling patterns
- File I/O operations
- String formatting and ANSI color codes
- Regular expressions
- Time operations and tickers

## Performance Benchmarks

Typical execution times on MacBook M2:
- **Local network devices**: 30-100ms
- **US-based services**: 100-300ms
- **International services**: 900ms-2s
- **Ping checks**: 600-700ms (average of 3 pings)
- **Total runtime**: ~2s (9 services checked concurrently)

## Project Roadmap

### Completed ✅
- [x] Basic HTTP/HTTPS health checks
- [x] Concurrent execution with goroutines
- [x] Individual response time measurements
- [x] Data structures with slices and structs
- [x] Periodic monitoring with configurable intervals
- [x] ICMP ping support for non-HTTP devices
- [x] Color-coded terminal output
- [x] Timestamped results
- [x] Persistent logging to file
- [x] Success rate and performance statistics

### Future Enhancements
- [ ] Email/SMS notifications for service failures
- [ ] Slack/Discord webhook integration
- [ ] Web dashboard for real-time visualization
- [ ] Historical data and trend analysis
- [ ] Configurable alert thresholds
- [ ] Multi-region latency testing
- [ ] JSON/CSV export for reporting
- [ ] Systemd/launchd service installation
- [ ] Configuration file support (YAML/JSON)
- [ ] API endpoints for programmatic access

## Use Cases

- **Home Network Monitoring**: Track router, NAS, smart devices, and IoT endpoints
- **Website Uptime Monitoring**: Monitor personal or business websites
- **Infrastructure Health**: Track critical services and APIs
- **Network Troubleshooting**: Identify connectivity issues and latency problems
- **Performance Baselines**: Establish normal response time patterns
- **Learning Go**: Educational project demonstrating production-ready Go patterns

## Built With

- [Go](https://golang.org/) - The Go Programming Language (1.23+)
- Standard Library - No external dependencies required
- System utilities - Native `ping` command for ICMP checks

## Author

**George Maddaloni**  
CTO of Operations, Mastercard Networks  
[www.georgemaddaloni.com](https://www.georgemaddaloni.com)

Professional background in global network infrastructure, security operations, and enterprise technology platforms supporting mission-critical payment processing.

## License

This project is open source and available for personal and educational use.

## Learning Journey

This project was created as a hands-on learning experience with Go programming, progressing through:

### Phase 1: Foundations
- Go installation and development environment setup
- Basic syntax, variables, functions, and types
- Error handling patterns
- HTTP client operations
- TLS certificate handling

### Phase 2: Data Structures
- Struct definitions for complex data modeling
- Slices for dynamic collections
- Multiple return values
- Type definitions and constants

### Phase 3: Concurrency (Go's Superpower) ✨
- Goroutines for parallel execution
- Channels for safe data sharing
- WaitGroups for synchronization
- Understanding concurrency vs parallelism
- Real-world performance optimization

### Phase 4: Production Features
- Periodic scheduling with tickers
- File I/O and persistent logging
- ANSI color codes for terminal output
- Statistical calculations
- Regular expressions for data parsing
- System command execution

### Phase 5: Professional Polish
- Code organization and modularity
- Comprehensive documentation
- Version control with Git
- Public repository management
- Real-world deployment considerations

---

## Quick Start Guide

**1. Install Go:** Download from [golang.org](https://golang.org)

**2. Clone and run:**
```bash
git clone https://github.com/MaddaloniG3/home-health-monitor.git
cd home-health-monitor
go run main.go
```

**3. Customize services in `main.go`**

**4. Monitor your network!** 🚀

---

**Project Status:** Production Ready ✅  
**Current Version:** 1.0  
**Last Updated:** December 28, 2024