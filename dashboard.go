package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type DataPoint struct {
	Timestamp    time.Time
	ResponseTime int64
}

type DashboardData struct {
	LastUpdate     string
	TotalEndpoints int
	Summary        []EndpointSummary
	TimeSeriesJSON template.JS
}

type EndpointSummary struct {
	Name         string
	Location     string
	Provider     string
	TestType     string
	LatestMs     int64
	AvgMs        int64
	MinMs        int64
	MaxMs        int64
	Count        int
	Status       string
	TrendPercent float64
}

func main() {
	http.HandleFunc("/", dashboardHandler)
	http.HandleFunc("/api/data", dataAPIHandler)

	fmt.Println("🌐 Cloud Latency Dashboard starting...")
	fmt.Println("📊 Open your browser to: http://localhost:8080")
	fmt.Println("Press Ctrl+C to stop")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	data, err := loadDashboardData()
	if err != nil {
		log.Printf("Error loading data: %v", err)
		http.Error(w, "Error loading data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}

	tmpl, err := template.New("dashboard").Funcs(funcMap).Parse(dashboardHTML)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execute error: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

func dataAPIHandler(w http.ResponseWriter, r *http.Request) {
	data, err := loadDashboardData()
	if err != nil {
		http.Error(w, "Error loading data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func loadDashboardData() (*DashboardData, error) {
	fileData, err := os.ReadFile("latency_history.json")
	if err != nil {
		return nil, err
	}

	var history map[string][]DataPoint
	if err := json.Unmarshal(fileData, &history); err != nil {
		return nil, err
	}

	var summary []EndpointSummary

	for serviceName, dataPoints := range history {
		if len(dataPoints) == 0 {
			continue
		}

		location, provider, testType := parseServiceName(serviceName)

		var totalMs int64
		minMs := int64(999999999)
		maxMs := int64(0)

		for _, point := range dataPoints {
			ms := point.ResponseTime / 1000000
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

		trendPct := 0.0
		if firstMs > 0 {
			trendPct = float64(latestMs-firstMs) / float64(firstMs) * 100
		}

		status := "steady"
		if trendPct > 50 {
			status = "slow"
		} else if trendPct < -50 {
			status = "fast"
		}

		summary = append(summary, EndpointSummary{
			Name:         serviceName,
			Location:     location,
			Provider:     provider,
			TestType:     testType,
			LatestMs:     latestMs,
			AvgMs:        avgMs,
			MinMs:        minMs,
			MaxMs:        maxMs,
			Count:        len(dataPoints),
			Status:       status,
			TrendPercent: trendPct,
		})
	}

	sort.Slice(summary, func(i, j int) bool {
		if summary[i].Location == summary[j].Location {
			return summary[i].TestType < summary[j].TestType
		}
		return summary[i].Location < summary[j].Location
	})

	// Don't use template.JS - just pass the raw JSON string
	timeSeriesBytes, err := json.Marshal(summary)
	if err != nil {
		return nil, err
	}

	return &DashboardData{
		LastUpdate:     time.Now().Format("2006-01-02 15:04:05"),
		TotalEndpoints: len(summary),
		Summary:        summary,
		TimeSeriesJSON: template.JS(timeSeriesBytes), // Pass bytes directly, not string
	}, nil
}

func parseServiceName(name string) (location, provider, testType string) {
	testType = "other"
	provider = "N/A"
	location = name

	if idx := strings.LastIndex(name, " - "); idx >= 0 {
		testType = strings.ToLower(name[idx+3:])
		name = name[:idx]
	}

	if start := strings.Index(name, "["); start >= 0 {
		if end := strings.Index(name[start:], "]"); end >= 0 {
			provider = name[start+1 : start+end]
			location = strings.TrimSpace(name[:start])
		}
	}

	return location, provider, testType
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cloud Latency Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333;
            min-height: 100vh;
            padding: 20px;
        }
        .container { max-width: 1400px; margin: 0 auto; }
        header {
            background: white;
            padding: 30px;
            border-radius: 15px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            margin-bottom: 30px;
        }
        h1 { color: #667eea; font-size: 2.5em; margin-bottom: 10px; }
        .subtitle { color: #666; font-size: 1.1em; }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: white;
            padding: 25px;
            border-radius: 12px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
        }
        .stat-label {
            color: #666;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 10px;
        }
        .stat-value { font-size: 2.5em; font-weight: bold; color: #667eea; }
        .chart-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
            gap: 30px;
            margin-bottom: 30px;
        }
        .chart-container {
            background: white;
            padding: 25px;
            border-radius: 12px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
        }
        .chart-title { font-size: 1.3em; margin-bottom: 20px; color: #333; }
        .table-container {
            background: white;
            padding: 25px;
            border-radius: 12px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
            overflow-x: auto;
        }
        table { width: 100%; border-collapse: collapse; }
        th {
            background: #667eea;
            color: white;
            padding: 15px;
            text-align: left;
            font-weight: 600;
        }
        td { padding: 12px 15px; border-bottom: 1px solid #e0e0e0; }
        tr:hover { background: #f5f5f5; }
        .status-badge {
            padding: 5px 12px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: 600;
            display: inline-block;
        }
        .status-fast { background: #4caf50; color: white; }
        .status-slow { background: #f44336; color: white; }
        .status-steady { background: #ff9800; color: white; }
        .test-type-ping { color: #2196f3; font-weight: 600; }
        .test-type-dns { color: #4caf50; font-weight: 600; }
        .test-type-http { color: #ff9800; font-weight: 600; }
        .refresh-info { text-align: center; color: white; margin-top: 20px; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🌐 Cloud Infrastructure Latency Dashboard</h1>
            <p class="subtitle">Real-time monitoring of AWS global endpoints | Last update: {{.LastUpdate}}</p>
        </header>
        
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Total Endpoints</div>
                <div class="stat-value">{{.TotalEndpoints}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Fastest Ping</div>
                <div class="stat-value" id="fastest-ping">--</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Slowest Ping</div>
                <div class="stat-value" id="slowest-ping">--</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Avg Latency</div>
                <div class="stat-value" id="avg-latency">--</div>
            </div>
        </div>
        
        <div class="chart-grid">
            <div class="chart-container">
                <h3 class="chart-title">Latency by Region</h3>
                <canvas id="regionChart"></canvas>
            </div>
            <div class="chart-container">
                <h3 class="chart-title">Test Type Comparison</h3>
                <canvas id="testTypeChart"></canvas>
            </div>
        </div>
        
        <div class="chart-container" style="margin-bottom: 30px;">
            <h3 class="chart-title">Geographic Distribution</h3>
            <canvas id="geoChart"></canvas>
        </div>
        
        <div class="table-container">
            <h3 class="chart-title">All Endpoints</h3>
            <table>
                <thead>
                    <tr>
                        <th>Location</th><th>Provider</th><th>Test Type</th>
                        <th>Latest (ms)</th><th>Avg (ms)</th><th>Min (ms)</th><th>Max (ms)</th>
                        <th>Samples</th><th>Trend</th><th>Status</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Summary}}
                    <tr>
                        <td>{{.Location}}</td>
                        <td>{{.Provider}}</td>
                        <td class="test-type-{{.TestType}}">{{.TestType}}</td>
                        <td>{{.LatestMs}}</td>
                        <td>{{.AvgMs}}</td>
                        <td>{{.MinMs}}</td>
                        <td>{{.MaxMs}}</td>
                        <td>{{.Count}}</td>
                        <td>{{printf "%.1f" .TrendPercent}}%</td>
                        <td><span class="status-badge status-{{.Status}}">{{.Status}}</span></td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        
        <div class="refresh-info">Dashboard auto-refreshes every 30 seconds</div>
    </div>
    
    <script>
        let timeSeriesData = [];
        try {
            timeSeriesData = {{.TimeSeriesJSON}};
            console.log('Loaded data points:', timeSeriesData.length);
        } catch (e) {
            console.error('Failed to parse data:', e);
        }
        
        const pingData = timeSeriesData.filter(d => d.TestType === 'ping');
        if (pingData.length > 0) {
            const latencies = pingData.map(d => d.AvgMs);
            document.getElementById('fastest-ping').textContent = Math.min(...latencies) + 'ms';
            document.getElementById('slowest-ping').textContent = Math.max(...latencies) + 'ms';
            document.getElementById('avg-latency').textContent = Math.round(latencies.reduce((a,b) => a+b, 0) / latencies.length) + 'ms';
        }
        
        const awsData = timeSeriesData.filter(d => d.Provider === 'AWS');
        const regionData = {};
        awsData.forEach(d => {
            if (!regionData[d.Location]) regionData[d.Location] = [];
            regionData[d.Location].push(d.AvgMs);
        });
        
        const regionLabels = Object.keys(regionData).sort().slice(0, 15);
        const regionValues = regionLabels.map(loc => Math.round(regionData[loc].reduce((a,b) => a+b, 0) / regionData[loc].length));
        
        new Chart(document.getElementById('regionChart'), {
            type: 'bar',
            data: {
                labels: regionLabels,
                datasets: [{ label: 'Avg Latency (ms)', data: regionValues, backgroundColor: 'rgba(102, 126, 234, 0.8)' }]
            },
            options: {
                responsive: true,
                plugins: { legend: { display: false } },
                scales: { y: { beginAtZero: true }, x: { ticks: { maxRotation: 45, minRotation: 45, font: { size: 10 } } } }
            }
        });
        
        const testTypes = { ping: [], dns: [], http: [] };
        awsData.forEach(d => { if (testTypes[d.TestType]) testTypes[d.TestType].push(d.AvgMs); });
        const testAvgs = Object.keys(testTypes).map(t => {
            const v = testTypes[t];
            return v.length > 0 ? Math.round(v.reduce((a,b) => a+b, 0) / v.length) : 0;
        });
        
        new Chart(document.getElementById('testTypeChart'), {
            type: 'doughnut',
            data: {
                labels: ['PING', 'DNS', 'HTTP'],
                datasets: [{ data: testAvgs, backgroundColor: ['rgba(33,150,243,0.8)', 'rgba(76,175,80,0.8)', 'rgba(255,152,0,0.8)'] }]
            },
            options: { responsive: true, plugins: { legend: { position: 'bottom' } } }
        });
        
        const continents = {
            'North America': ['Ashburn', 'Columbus', 'San Jose', 'Portland', 'Montreal'],
            'South America': ['Paulo'],
            'Europe': ['London', 'Paris', 'Frankfurt', 'Stockholm', 'Milan'],
            'Middle East': ['Dubai', 'Riyadh'],
            'Asia': ['Mumbai', 'Hyderabad', 'Singapore', 'Jakarta', 'Tokyo', 'Seoul', 'Osaka'],
            'Africa': ['Cape Town'],
            'Oceania': ['Sydney', 'Melbourne']
        };
        
        const continentAvgs = {};
        Object.keys(continents).forEach(cont => {
            const vals = awsData.filter(d => continents[cont].some(c => d.Location.includes(c))).map(d => d.AvgMs);
            if (vals.length > 0) continentAvgs[cont] = Math.round(vals.reduce((a,b) => a+b, 0) / vals.length);
        });
        
        new Chart(document.getElementById('geoChart'), {
            type: 'bar',
            data: {
                labels: Object.keys(continentAvgs),
                datasets: [{ label: 'Avg Latency (ms)', data: Object.values(continentAvgs), backgroundColor: 'rgba(118, 75, 162, 0.8)' }]
            },
            options: {
                responsive: true,
                indexAxis: 'y',
                plugins: { legend: { display: false } },
                scales: { x: { beginAtZero: true } }
            }
        });
        
        setTimeout(() => location.reload(), 30000);
    </script>
</body>
</html>`
