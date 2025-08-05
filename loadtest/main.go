package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Event structure matching the API
type Event struct {
	EventTimestamp int64  `json:"eventtimestamp"`
	EventTime      string `json:"eventtime"`
	ID             string `json:"id"`
	Domain         string `json:"domain"`
	Subdomain      string `json:"subdomain"`
	Code           string `json:"code"`
	Version        string `json:"version"`
	BranchID       int    `json:"branchid"`
	ChannelID      int    `json:"channelid"`
	CustomerID     int    `json:"customerid"`
	UserID         int    `json:"userid"`
	Payload        string `json:"payload"`
}

// EventResponse structure matching the API
type EventResponse struct {
	SuccessEventIds []string `json:"successEventIds"`
	InvalidEventIds []string `json:"invalidEventIds"`
	FailedEventIds  []string `json:"failedEventIds"`
}

// Statistics for load test results
type LoadTestStats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TimeoutRequests int64
	TotalEvents     int64
	SuccessEvents   int64
	FailedEvents    int64
	InvalidEvents   int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	StartTime       time.Time
	EndTime         time.Time
}

var (
	// Command line flags
	duration     = flag.Int("duration", 30, "Test duration in seconds")
	goroutines   = flag.Int("goroutines", 10, "Number of concurrent goroutines")
	apiURL       = flag.String("url", "http://localhost:8080", "API base URL")
	eventsPerReq = flag.Int("events", 1, "Number of events per request")
	requestDelay = flag.Int("delay", 100, "Delay between requests in milliseconds")
	verbose      = flag.Bool("verbose", false, "Verbose output")

	// Statistics
	stats = &LoadTestStats{
		MinLatency: time.Hour, // Start with a high value
	}
	statsMutex sync.Mutex
)

// Generate a random event
func generateRandomEvent() Event {
	domains := []string{"Banking"}
	subdomains := []string{"Domestic"}
	codes := []string{"Created"}

	now := time.Now()

	return Event{
		EventTimestamp: now.UnixNano(),
		EventTime:      now.Format("2006-01-02T15:04:05.000Z07:00"),
		ID:             fmt.Sprintf("load-test-%d-%d", rand.Intn(100000), now.UnixNano()),
		Domain:         domains[rand.Intn(len(domains))],
		Subdomain:      subdomains[rand.Intn(len(subdomains))],
		Code:           codes[rand.Intn(len(codes))],
		Version:        "1.0",
		BranchID:       rand.Intn(9000) + 1000,
		ChannelID:      rand.Intn(100),
		CustomerID:     rand.Intn(1000000) + 100000,
		UserID:         rand.Intn(100000) + 10000,
		Payload:        fmt.Sprintf("Load test payload %d", rand.Intn(10000)),
	}
}

// Send a single request to the API
func sendRequest(client *http.Client, events []Event) (*EventResponse, time.Duration, error) {
	jsonData, err := json.Marshal(events)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal events: %w", err)
	}

	start := time.Now()

	resp, err := client.Post(*apiURL+"/events", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, time.Since(start), fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, latency, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var response EventResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, latency, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, latency, nil
}

// Update statistics
func updateStats(response *EventResponse, latency time.Duration, err error) {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	atomic.AddInt64(&stats.TotalRequests, 1)

	if err != nil {
		// Check if it's a timeout error
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			atomic.AddInt64(&stats.TimeoutRequests, 1)
			if *verbose {
				log.Printf("Request timeout: %v", err)
			}
		} else {
			atomic.AddInt64(&stats.FailedRequests, 1)
			if *verbose {
				log.Printf("Request failed: %v", err)
			}
		}
		return
	}

	atomic.AddInt64(&stats.SuccessRequests, 1)

	// Update event statistics
	if response != nil {
		atomic.AddInt64(&stats.TotalEvents, int64(len(response.SuccessEventIds)+len(response.FailedEventIds)+len(response.InvalidEventIds)))
		atomic.AddInt64(&stats.SuccessEvents, int64(len(response.SuccessEventIds)))
		atomic.AddInt64(&stats.FailedEvents, int64(len(response.FailedEventIds)))
		atomic.AddInt64(&stats.InvalidEvents, int64(len(response.InvalidEventIds)))
	}

	// Update latency statistics
	stats.TotalLatency += latency
	if latency < stats.MinLatency {
		stats.MinLatency = latency
	}
	if latency > stats.MaxLatency {
		stats.MaxLatency = latency
	}

	if *verbose && response != nil {
		log.Printf("Request successful - Success: %d, Failed: %d, Invalid: %d, Latency: %v",
			len(response.SuccessEventIds), len(response.FailedEventIds), len(response.InvalidEventIds), latency)
	}
}

// Worker goroutine
func worker(workerID int, stopChan <-chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 1 * time.Second, // 1 second timeout for REST API calls
	}

	log.Printf("Worker %d started", workerID)

	for {
		select {
		case <-stopChan:
			log.Printf("Worker %d stopped", workerID)
			return
		default:
			// Generate events for this request
			events := make([]Event, *eventsPerReq)
			for i := 0; i < *eventsPerReq; i++ {
				events[i] = generateRandomEvent()
			}

			// Send request
			response, latency, err := sendRequest(client, events)
			updateStats(response, latency, err)

			// Wait before next request
			if *requestDelay > 0 {
				time.Sleep(time.Duration(*requestDelay) * time.Millisecond)
			}
		}
	}
}

// Print real-time statistics
func printRealTimeStats(stopChan <-chan bool) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			printCurrentStats()
		}
	}
}

// Print current statistics
func printCurrentStats() {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	elapsed := time.Since(stats.StartTime)
	requestsPerSecond := float64(stats.TotalRequests) / elapsed.Seconds()
	eventsPerSecond := float64(stats.TotalEvents) / elapsed.Seconds()

	var avgLatency time.Duration
	if stats.SuccessRequests > 0 {
		avgLatency = stats.TotalLatency / time.Duration(stats.SuccessRequests)
	}

	fmt.Printf("\n=== Load Test Statistics (Elapsed: %v) ===\n", elapsed.Round(time.Second))
	fmt.Printf("Requests: %d total, %d success, %d failed, %d timeout (%.2f req/sec)\n",
		stats.TotalRequests, stats.SuccessRequests, stats.FailedRequests, stats.TimeoutRequests, requestsPerSecond)
	fmt.Printf("Events: %d total, %d success, %d failed, %d invalid (%.2f events/sec)\n",
		stats.TotalEvents, stats.SuccessEvents, stats.FailedEvents, stats.InvalidEvents, eventsPerSecond)
	fmt.Printf("Latency: avg=%v, min=%v, max=%v\n",
		avgLatency.Round(time.Millisecond),
		stats.MinLatency.Round(time.Millisecond),
		stats.MaxLatency.Round(time.Millisecond))
	fmt.Printf("Success Rate: %.2f%% (requests), %.2f%% (events)\n",
		float64(stats.SuccessRequests)/float64(stats.TotalRequests)*100,
		float64(stats.SuccessEvents)/float64(stats.TotalEvents)*100)
	if stats.TimeoutRequests > 0 {
		fmt.Printf("Timeout Rate: %.2f%% (%d/%d requests)\n",
			float64(stats.TimeoutRequests)/float64(stats.TotalRequests)*100,
			stats.TimeoutRequests, stats.TotalRequests)
	}
}

// Print final report
func printFinalReport() {
	separator := strings.Repeat("=", 80)
	fmt.Printf("\n%s\n", separator)
	fmt.Printf("                         LOAD TEST FINAL REPORT\n")
	fmt.Printf("%s\n", separator)

	statsMutex.Lock()
	defer statsMutex.Unlock()

	totalDuration := stats.EndTime.Sub(stats.StartTime)
	requestsPerSecond := float64(stats.TotalRequests) / totalDuration.Seconds()
	eventsPerSecond := float64(stats.TotalEvents) / totalDuration.Seconds()

	var avgLatency time.Duration
	if stats.SuccessRequests > 0 {
		avgLatency = stats.TotalLatency / time.Duration(stats.SuccessRequests)
	}

	fmt.Printf("Test Configuration:\n")
	fmt.Printf("  Duration: %d seconds\n", *duration)
	fmt.Printf("  Goroutines: %d\n", *goroutines)
	fmt.Printf("  Events per request: %d\n", *eventsPerReq)
	fmt.Printf("  Request delay: %d ms\n", *requestDelay)
	fmt.Printf("  API URL: %s\n", *apiURL)
	fmt.Printf("\n")

	fmt.Printf("Request Statistics:\n")
	fmt.Printf("  Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("  Successful requests: %d\n", stats.SuccessRequests)
	fmt.Printf("  Failed requests: %d\n", stats.FailedRequests)
	fmt.Printf("  Timeout requests: %d\n", stats.TimeoutRequests)
	fmt.Printf("  Requests per second: %.2f\n", requestsPerSecond)
	fmt.Printf("  Success rate: %.2f%%\n", float64(stats.SuccessRequests)/float64(stats.TotalRequests)*100)
	if stats.TimeoutRequests > 0 {
		fmt.Printf("  Timeout rate: %.2f%%\n", float64(stats.TimeoutRequests)/float64(stats.TotalRequests)*100)
	}
	fmt.Printf("\n")

	fmt.Printf("Event Statistics:\n")
	fmt.Printf("  Total events: %d\n", stats.TotalEvents)
	fmt.Printf("  Successful events: %d\n", stats.SuccessEvents)
	fmt.Printf("  Failed events: %d\n", stats.FailedEvents)
	fmt.Printf("  Invalid events: %d\n", stats.InvalidEvents)
	fmt.Printf("  Events per second: %.2f\n", eventsPerSecond)
	fmt.Printf("  Event success rate: %.2f%%\n", float64(stats.SuccessEvents)/float64(stats.TotalEvents)*100)
	fmt.Printf("\n")

	fmt.Printf("Latency Statistics:\n")
	fmt.Printf("  Average latency: %v\n", avgLatency.Round(time.Millisecond))
	fmt.Printf("  Minimum latency: %v\n", stats.MinLatency.Round(time.Millisecond))
	fmt.Printf("  Maximum latency: %v\n", stats.MaxLatency.Round(time.Millisecond))
	fmt.Printf("\n")

	fmt.Printf("%s\n", separator)
}

func main() {
	flag.Parse()

	// Modern Go random number generation (no need for seed)
	// rand.Seed is deprecated since Go 1.20

	fmt.Printf("Starting load test with %d goroutines for %d seconds...\n", *goroutines, *duration)
	fmt.Printf("Target API: %s\n", *apiURL)
	fmt.Printf("Events per request: %d\n", *eventsPerReq)
	fmt.Printf("Request delay: %d ms\n", *requestDelay)
	fmt.Printf("Verbose mode: %t\n", *verbose)

	// Test API connectivity first
	fmt.Printf("\nTesting API connectivity...\n")
	client := &http.Client{Timeout: 1 * time.Second} // 1 second timeout for health check
	resp, err := client.Get(*apiURL + "/protected/health")
	if err != nil {
		log.Fatalf("Failed to connect to API: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API health check failed with status: %d", resp.StatusCode)
	}
	fmt.Printf("✓ API connectivity test passed\n")

	// Initialize statistics
	stats.StartTime = time.Now()

	// Create stop channel and wait group
	stopChan := make(chan bool)
	var wg sync.WaitGroup

	// Start real-time statistics printer
	go printRealTimeStats(stopChan)

	// Start worker goroutines
	fmt.Printf("\nStarting %d worker goroutines...\n", *goroutines)
	for i := 0; i < *goroutines; i++ {
		wg.Add(1)
		go worker(i+1, stopChan, &wg)
	}

	// Wait for specified duration
	fmt.Printf("Load test running for %d seconds...\n", *duration)
	time.Sleep(time.Duration(*duration) * time.Second)

	// Stop all workers
	fmt.Printf("\nStopping load test...\n")
	close(stopChan)
	wg.Wait()

	stats.EndTime = time.Now()

	// Print final report
	printFinalReport()
}
