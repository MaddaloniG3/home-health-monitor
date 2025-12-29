# Home Network Health Monitor

A lightweight network monitoring tool written in Go that checks the health and connectivity of network services and endpoints.

## Features

- ✅ HTTP/HTTPS endpoint monitoring
- ✅ Support for self-signed certificates (local devices)
- ✅ Latency measurements for remote endpoints
- ✅ Simple, extensible configuration
- ✅ Fast execution with clean output

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

The monitor checks a predefined list of services and reports their status:
```bash
go run main.go
```

Example output:
```
=== Network Health Monitor ===
Checking Home Router at https://192.168.3.1...
[UP] Home Router

Checking Google at https://www.google.com...
[UP] Google

Checking Cape Town, SA (UCT) at https://www.uct.ac.za...
[UP] Cape Town, SA (UCT) - Latency: 1.521541667s
```

## Configuration

To monitor your own services, edit the `services` slice in `main.go`:
```go
services := []Service{
    {Name: "My Router", URL: "https://192.168.1.1", Insecure: true},
    {Name: "My Server", URL: "https://myserver.local", Insecure: false},
}
```

**Service struct fields:**
- `Name`: Display name for the service
- `URL`: Full URL to check (http:// or https://)
- `Insecure`: Set to `true` for local devices with self-signed certificates

## Roadmap

- [ ] Concurrent health checks for faster execution
- [ ] Periodic monitoring with configurable intervals
- [ ] Logging results to file
- [ ] ICMP ping support for non-HTTP devices
- [ ] Email/Slack notifications for downtime
- [ ] Web dashboard for visualization

## Built With

- [Go](https://golang.org/) - The Go Programming Language
- Standard library packages: `net/http`, `crypto/tls`, `time`

## Author

George Maddaloni

## License

This project is open source and available for personal and educational use.

## Learning Journey

This project was created as part of learning Go programming, with a focus on:
- Go basics (variables, functions, structs)
- Working with slices and data structures
- HTTP client operations
- TLS/SSL certificate handling
- Error handling patterns in Go