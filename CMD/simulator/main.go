package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	userAgents = []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.6367.82 Mobile Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPad; CPU OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/124.0.6367.88 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"Twitterbot/1.0",
	}

	referrers = []string{
		"https://t.co/rEaL7R4nD0",
		"https://www.google.com/",
		"https://l.facebook.com/",
		"https://www.linkedin.com/feed/",
		"https://www.reddit.com/r/programming/",
		"https://news.ycombinator.com/",
		"", // Direct navigation
	}

	ipSubnets = []string{
		"104.28.",  // Cloudflare
		"172.56.",  // T-Mobile US
		"24.120.",  // Comcast
		"185.220.", // EU Transit
		"110.54.",  // APAC
		"177.18.",  // LATAM
		"82.165.",  // Ionos / Germany
		"140.213.", // Telkom / SE Asia
	}
)

func randomIP() string {
	prefix := ipSubnets[rand.Intn(len(ipSubnets))]
	return fmt.Sprintf("%s%d.%d", prefix, rand.Intn(254)+1, rand.Intn(254)+1)
}

func randomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

func randomReferrer() string {
	return referrers[rand.Intn(len(referrers))]
}

type HTTPResult struct {
	StatusCode int
	Duration   time.Duration
	Err        error
	RouteType  string
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Do not follow redirects so we measure the 302 hot path latency
			return http.ErrUseLastResponse
		},
	}
}

func main() {
	baseURL := flag.String("base-url", "http://localhost:8080", "Target URL Shortener API base URL")
	mode := flag.String("mode", "all", "Execution mode: 'smoke', 'load', or 'all'")
	concurrency := flag.Int("concurrency", 50, "Number of concurrent virtual user workers for load test")
	totalRequests := flag.Int("requests", 5000, "Total number of simulated user requests for load test")
	clientJitterMs := flag.Int("client-jitter-ms", 0, "Artificial client-side WAN latency jitter (ms)")
	flag.Parse()

	fmt.Println("================================================================================")
	fmt.Printf("100M-Systems URL Shortener: Enterprise Scale & Smoke Testing Suite\n")
	fmt.Printf("Target Base URL : %s\n", *baseURL)
	fmt.Printf("Execution Mode  : %s\n", *mode)
	fmt.Println("================================================================================")

	client := newHTTPClient()

	// 1. Run Smoke Tests if requested
	if *mode == "smoke" || *mode == "all" {
		runSmokeTestSuite(client, *baseURL)
	}

	// 2. Run 100M Scale Load Simulation if requested
	if *mode == "load" || *mode == "all" {
		run100MLoadSimulation(client, *baseURL, *concurrency, *totalRequests, *clientJitterMs)
	}
}

// ==============================================================================
// 1. COMPREHENSIVE SMOKE TEST SUITE
// ==============================================================================

type smokeTestStep struct {
	Name string
	Run  func() error
}

func runSmokeTestSuite(client *http.Client, baseURL string) {
	fmt.Println("\n>>> [PHASE 1] RUNNING COMPREHENSIVE ENDPOINT SMOKE TESTS")
	fmt.Println("--------------------------------------------------------------------------------")

	var createdCode string
	targetURL := "https://github.com/segmentio/kafka-go"

	tests := []smokeTestStep{
		{
			Name: "1. Health & Prometheus /metrics endpoint",
			Run: func() error {
				resp, err := client.Get(baseURL + "/metrics")
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("expected 200, got %d", resp.StatusCode)
				}
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), "http_requests_in_flight") && !strings.Contains(string(body), "go_info") {
					return fmt.Errorf("metrics output missing Prometheus descriptors")
				}
				return nil
			},
		},
		{
			Name: "2. POST /api/shorten (Valid URL)",
			Run: func() error {
				payload := map[string]string{"url": targetURL}
				data, _ := json.Marshal(payload)
				req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/shorten", bytes.NewReader(data))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Forwarded-For", "203.0.113.195")

				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusCreated {
					return fmt.Errorf("expected 201, got %d", resp.StatusCode)
				}

				var result struct {
					Code        string `json:"code"`
					ShortURL    string `json:"short_url"`
					OriginalURL string `json:"original_url"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					return err
				}
				if len(result.Code) != 8 || result.OriginalURL != targetURL {
					return fmt.Errorf("invalid response payload: %+v", result)
				}
				createdCode = result.Code
				return nil
			},
		},
		{
			Name: "3. POST /api/shorten (Invalid URL -> 400 Bad Request)",
			Run: func() error {
				payload := map[string]string{"url": "invalid-url-scheme"}
				data, _ := json.Marshal(payload)
				req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/shorten", bytes.NewReader(data))
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusBadRequest {
					return fmt.Errorf("expected 400, got %d", resp.StatusCode)
				}
				return nil
			},
		},
		{
			Name: "4. POST /api/shorten (Empty Body / Malformed JSON -> 400 Bad Request)",
			Run: func() error {
				req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/shorten", strings.NewReader("{bad_json"))
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusBadRequest {
					return fmt.Errorf("expected 400, got %d", resp.StatusCode)
				}
				return nil
			},
		},
		{
			Name: "5. GET /{code} (Redirect Hot Path -> 302 Found)",
			Run: func() error {
				if createdCode == "" {
					return fmt.Errorf("no code created in previous step")
				}
				req, _ := http.NewRequest(http.MethodGet, baseURL+"/"+createdCode, nil)
				req.Header.Set("X-Forwarded-For", "198.51.100.42")
				req.Header.Set("User-Agent", "Mozilla/5.0 SmokeTestAgent/1.0")
				req.Header.Set("Referer", "https://google.com")

				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusFound {
					return fmt.Errorf("expected 302, got %d", resp.StatusCode)
				}
				loc := resp.Header.Get("Location")
				if loc != targetURL {
					return fmt.Errorf("expected Location %s, got %s", targetURL, loc)
				}
				return nil
			},
		},
		{
			Name: "6. GET /{code} (Unknown Code -> 404 Not Found)",
			Run: func() error {
				req, _ := http.NewRequest(http.MethodGet, baseURL+"/UNKNOWN1", nil)
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusNotFound {
					return fmt.Errorf("expected 404, got %d", resp.StatusCode)
				}
				return nil
			},
		},
		{
			Name: "7. GET /api/urls/{code} (Metadata Inspection -> 200 OK)",
			Run: func() error {
				if createdCode == "" {
					return fmt.Errorf("no code created")
				}
				resp, err := client.Get(baseURL + "/api/urls/" + createdCode)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("expected 200, got %d", resp.StatusCode)
				}
				var record struct {
					Code     string `json:"code"`
					Original string `json:"original_url"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
					return err
				}
				if record.Code != createdCode || record.Original != targetURL {
					return fmt.Errorf("mismatch in metadata: %+v", record)
				}
				return nil
			},
		},
		{
			Name: "8. GET /api/urls/{code}/analytics (Click Stream Analytics -> 200 OK)",
			Run: func() error {
				if createdCode == "" {
					return fmt.Errorf("no code created")
				}
				resp, err := client.Get(baseURL + "/api/urls/" + createdCode + "/analytics")
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("expected 200, got %d", resp.StatusCode)
				}
				var summary struct {
					Code        string `json:"code"`
					OriginalURL string `json:"original_url"`
					TotalClicks int64  `json:"total_clicks"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
					return err
				}
				if summary.Code != createdCode {
					return fmt.Errorf("unexpected analytics code: %s", summary.Code)
				}
				return nil
			},
		},
		{
			Name: "9. OPTIONS /api/shorten (CORS Preflight Headers)",
			Run: func() error {
				req, _ := http.NewRequest(http.MethodOptions, baseURL+"/api/shorten", nil)
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
					return fmt.Errorf("expected 204/200 on OPTIONS, got %d", resp.StatusCode)
				}
				origin := resp.Header.Get("Access-Control-Allow-Origin")
				if origin == "" {
					return fmt.Errorf("missing Access-Control-Allow-Origin header")
				}
				return nil
			},
		},
		{
			Name: "10. Rate Limiter Telemetry & Headers Check",
			Run: func() error {
				req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/urls/"+createdCode, nil)
				req.Header.Set("X-Forwarded-For", "198.51.100.99")
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				limitHeader := resp.Header.Get("X-RateLimit-Limit")
				if limitHeader == "" {
					return fmt.Errorf("missing X-RateLimit-Limit header")
				}
				return nil
			},
		},
		{
			Name: "11. Prometheus Telemetry Verification (Traffic counters active)",
			Run: func() error {
				resp, err := client.Get(baseURL + "/metrics")
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), "http_requests_total") {
					return fmt.Errorf("missing http_requests_total after traffic")
				}
				if !strings.Contains(string(body), "url_shortener_redirects_total") {
					return fmt.Errorf("missing url_shortener_redirects_total")
				}
				return nil
			},
		},
		{
			Name: "12. POST /api/shorten (Custom Vanity Alias & Conflict Check)",
			Run: func() error {
				alias := fmt.Sprintf("promo-%d", time.Now().UnixNano()%100000)
				data, _ := json.Marshal(map[string]any{
					"url":         "https://go.dev/doc/",
					"custom_code": alias,
				})
				req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/shorten", bytes.NewReader(data))
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusCreated {
					return fmt.Errorf("expected 201 on custom alias, got %d", resp.StatusCode)
				}

				// Try creating same alias again -> 409 Conflict
				req2, _ := http.NewRequest(http.MethodPost, baseURL+"/api/shorten", bytes.NewReader(data))
				req2.Header.Set("Content-Type", "application/json")
				resp2, err := client.Do(req2)
				if err != nil {
					return err
				}
				defer resp2.Body.Close()
				if resp2.StatusCode != http.StatusConflict {
					return fmt.Errorf("expected 409 on duplicate custom alias, got %d", resp2.StatusCode)
				}
				return nil
			},
		},
		{
			Name: "13. GET /api/urls/{code}/analytics (Dimensional Breakdowns)",
			Run: func() error {
				if createdCode == "" {
					return fmt.Errorf("no code created")
				}
				resp, err := client.Get(baseURL + "/api/urls/" + createdCode + "/analytics")
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				var summary struct {
					Browsers         map[string]int64 `json:"browsers"`
					OperatingSystems map[string]int64 `json:"operating_systems"`
					Devices          map[string]int64 `json:"devices"`
					TopReferrers     map[string]int64 `json:"top_referrers"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
					return err
				}
				if summary.Browsers == nil || summary.Devices == nil {
					return fmt.Errorf("missing dimensional breakdown maps in analytics response")
				}
				return nil
			},
		},
	}

	passedCount := 0
	for _, t := range tests {
		start := time.Now()
		err := t.Run()
		dur := time.Since(start)
		if err != nil {
			fmt.Printf("  [FAIL] %-55s (%v) -> ERROR: %v\n", t.Name, dur.Round(time.Microsecond), err)
		} else {
			fmt.Printf("  [PASS] %-55s (%v)\n", t.Name, dur.Round(time.Microsecond))
			passedCount++
		}
	}

	fmt.Printf("\nSmoke Test Summary: %d / %d Tests Passed (%.1f%%)\n",
		passedCount, len(tests), float64(passedCount)/float64(len(tests))*100)
	if passedCount != len(tests) {
		fmt.Println("Warning: Some smoke tests failed. Review the logs above.")
	} else {
		fmt.Println("All Endpoint Smoke Tests PASSED Successfully!")
	}
}

// ==============================================================================
// 2. 100M USERS SCALE BENCHMARK & TRAFFIC SIMULATION
// ==============================================================================

func run100MLoadSimulation(client *http.Client, baseURL string, concurrency, totalRequests, clientJitterMs int) {
	fmt.Println("\n>>> [PHASE 2] RUNNING 100M USERS TRAFFIC & DATA PROCESSING SIMULATION")
	fmt.Printf("Concurrency Workers : %d\n", concurrency)
	fmt.Printf("Total Requests      : %d\n", totalRequests)
	if clientJitterMs > 0 {
		fmt.Printf("Client WAN Jitter   : ~%d ms per round-trip\n", clientJitterMs)
	}
	fmt.Println("--------------------------------------------------------------------------------")

	// Step 1: Pre-seed a working catalog of URLs (simulating short link inventory)
	fmt.Print("Seeding active short URLs in database & cache... ")
	seededCodes := make([]string, 0, 30)
	for i := 1; i <= 30; i++ {
		target := fmt.Sprintf("https://system-design-100m.org/topic/%d/article-%d", i, rand.Intn(10000))
		data, _ := json.Marshal(map[string]string{"url": target})
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/shorten", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", randomIP())

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusCreated {
			var res struct{ Code string }
			_ = json.NewDecoder(resp.Body).Decode(&res)
			resp.Body.Close()
			if res.Code != "" {
				seededCodes = append(seededCodes, res.Code)
			}
		}
	}
	fmt.Printf("Ready! (%d short links seeded)\n", len(seededCodes))

	if len(seededCodes) == 0 {
		fmt.Println("Error: Failed to seed short URLs. Aborting load test.")
		return
	}

	// Hot URLs: Top 20% of links get 80% of click traffic (Zipf/Pareto principle)
	numHot := len(seededCodes) / 5
	if numHot < 1 {
		numHot = 1
	}
	hotCodes := seededCodes[:numHot]

	// Step 2: Concurrency worker pool
	type Job struct {
		ID        int
		RouteType string
		Method    string
		Path      string
		Body      []byte
	}

	jobs := make(chan Job, totalRequests)
	results := make(chan HTTPResult, totalRequests)

	// Populate Jobs according to 100M traffic distribution:
	// - 75% Redirects (GET /{code}) [80% hot URLs, 20% cold URLs]
	// - 10% Shorten (POST /api/shorten)
	// - 5% Metadata lookups (GET /api/urls/{code})
	// - 5% Analytics queries (GET /api/urls/{code}/analytics)
	// - 5% Cold cache / 404 lookups (GET /{unknown})
	for i := 0; i < totalRequests; i++ {
		roll := rand.Intn(100)
		var job Job
		job.ID = i

		switch {
		case roll < 75: // Redirect (Hot path)
			job.RouteType = "Redirect"
			job.Method = http.MethodGet
			var code string
			if rand.Intn(100) < 80 {
				code = hotCodes[rand.Intn(len(hotCodes))]
			} else {
				code = seededCodes[rand.Intn(len(seededCodes))]
			}
			job.Path = "/" + code

		case roll < 85: // Shorten URL
			job.RouteType = "Shorten"
			job.Method = http.MethodPost
			job.Path = "/api/shorten"
			target := fmt.Sprintf("https://random-destination-%d.com/page/%d", rand.Intn(50000), i)
			job.Body, _ = json.Marshal(map[string]string{"url": target})

		case roll < 90: // URL Metadata
			job.RouteType = "Metadata"
			job.Method = http.MethodGet
			code := seededCodes[rand.Intn(len(seededCodes))]
			job.Path = "/api/urls/" + code

		case roll < 95: // Analytics
			job.RouteType = "Analytics"
			job.Method = http.MethodGet
			code := hotCodes[rand.Intn(len(hotCodes))]
			job.Path = "/api/urls/" + code + "/analytics"

		default: // Cold / 404 Lookups
			job.RouteType = "404_Cold"
			job.Method = http.MethodGet
			job.Path = fmt.Sprintf("/miss%04d", rand.Intn(9999))
		}

		jobs <- job
	}
	close(jobs)

	// Step 3: Run benchmark workers
	var wg sync.WaitGroup
	var completedCount atomic.Int64

	fmt.Printf("Firing %d operations with %d parallel virtual user threads...\n", totalRequests, concurrency)
	simStart := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				var bodyReader io.Reader
				if len(job.Body) > 0 {
					bodyReader = bytes.NewReader(job.Body)
				}

				req, err := http.NewRequestWithContext(context.Background(), job.Method, baseURL+job.Path, bodyReader)
				if err != nil {
					results <- HTTPResult{Err: err, RouteType: job.RouteType}
					completedCount.Add(1)
					continue
				}

				// Inject unique simulated user identity
				req.Header.Set("X-Forwarded-For", randomIP())
				req.Header.Set("User-Agent", randomUserAgent())
				req.Header.Set("Referer", randomReferrer())
				if job.Method == http.MethodPost {
					req.Header.Set("Content-Type", "application/json")
				}

				// Optional client-side WAN jitter simulation
				if clientJitterMs > 0 {
					jitter := time.Duration(clientJitterMs/2+rand.Intn(clientJitterMs)) * time.Millisecond
					time.Sleep(jitter)
				}

				t0 := time.Now()
				resp, reqErr := client.Do(req)
				dur := time.Since(t0)

				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}

				results <- HTTPResult{
					StatusCode: statusCode,
					Duration:   dur,
					Err:        reqErr,
					RouteType:  job.RouteType,
				}

				completed := completedCount.Add(1)
				if completed%(int64(totalRequests)/5) == 0 || completed == int64(totalRequests) {
					pct := float64(completed) / float64(totalRequests) * 100
					rps := float64(completed) / time.Since(simStart).Seconds()
					fmt.Printf("  Progress: %6d / %d (%.0f%%) | Current Throughput: %8.1f req/sec\n",
						completed, totalRequests, pct, rps)
				}
			}
		}(w)
	}

	wg.Wait()
	close(results)
	totalDuration := time.Since(simStart)

	// Step 4: Aggregate and Calculate Latency Percentiles
	var latencies []time.Duration
	statusCounts := make(map[int]int)
	routeLatencies := make(map[string][]time.Duration)
	var totalErrors int

	for res := range results {
		if res.Err != nil {
			totalErrors++
			continue
		}
		latencies = append(latencies, res.Duration)
		statusCounts[res.StatusCode]++
		routeLatencies[res.RouteType] = append(routeLatencies[res.RouteType], res.Duration)
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	n := len(latencies)
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}

	var mean, p50, p90, p95, p99, minLat, maxLat time.Duration
	if n > 0 {
		mean = sum / time.Duration(n)
		minLat = latencies[0]
		maxLat = latencies[n-1]
		p50 = latencies[n*50/100]
		p90 = latencies[n*90/100]
		p95 = latencies[n*95/100]
		p99 = latencies[n*99/100]
	}

	rps := float64(n) / totalDuration.Seconds()
	successRate := float64(n-totalErrors) / float64(totalRequests) * 100

	// Step 5: Render Enterprise Performance Report
	fmt.Println("\n================================================================================")
	fmt.Println("              100M USERS REAL-WORLD BENCHMARK PERFORMANCE REPORT                ")
	fmt.Println("================================================================================")
	fmt.Printf("Total Elapsed Time   : %v\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("Total Requests Sent  : %d\n", totalRequests)
	fmt.Printf("Successful Requests  : %d (%.2f%%)\n", n, successRate)
	fmt.Printf("Network Failures     : %d\n", totalErrors)
	fmt.Printf("Overall Throughput   : %.2f requests/sec\n", rps)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("LATENCY PERCENTILE DISTRIBUTION (End-to-End Processing):")
	fmt.Printf("  Minimum Latency    : %v\n", minLat.Round(time.Microsecond))
	fmt.Printf("  Mean (Average)     : %v\n", mean.Round(time.Microsecond))
	fmt.Printf("  P50 (Median)       : %v\n", p50.Round(time.Microsecond))
	fmt.Printf("  P90 Percentile     : %v\n", p90.Round(time.Microsecond))
	fmt.Printf("  P95 Percentile     : %v\n", p95.Round(time.Microsecond))
	fmt.Printf("  P99 Percentile     : %v\n", p99.Round(time.Microsecond))
	fmt.Printf("  Maximum Latency    : %v\n", maxLat.Round(time.Microsecond))
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("LATENCY BREAKDOWN BY ROUTE TYPE:")
	for rType, list := range routeLatencies {
		sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
		cnt := len(list)
		rMean := time.Duration(0)
		for _, dur := range list {
			rMean += dur
		}
		if cnt > 0 {
			rMean /= time.Duration(cnt)
			rP50 := list[cnt*50/100]
			rP95 := list[cnt*95/100]
			fmt.Printf("  %-12s (%5d calls): Avg: %-8v | P50: %-8v | P95: %-8v\n",
				rType, cnt, rMean.Round(time.Microsecond), rP50.Round(time.Microsecond), rP95.Round(time.Microsecond))
		}
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("HTTP STATUS CODE DISTRIBUTION:")
	for code, cnt := range statusCounts {
		fmt.Printf("  HTTP %d : %6d requests (%.1f%%)\n", code, cnt, float64(cnt)/float64(n)*100)
	}
	fmt.Println("================================================================================")
}
