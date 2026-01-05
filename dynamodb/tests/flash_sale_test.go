package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// FlashSaleTestResult stores results for the flash sale simulation
type FlashSaleTestResult struct {
	Environment     string                 `json:"environment"` // "localstack" or "aws"
	TestName        string                 `json:"test_name"`
	Duration        float64                `json:"duration_seconds"`
	TotalUsers      int                    `json:"total_users"`
	TotalOperations int                    `json:"total_operations"`
	SuccessCount    int                    `json:"success_count"`
	FailureCount    int                    `json:"failure_count"`
	SuccessRate     float64                `json:"success_rate_percent"`
	TotalThroughput float64                `json:"total_throughput_ops_per_sec"`
	AvgResponseTime float64                `json:"avg_response_time_ms"`
	MinResponseTime float64                `json:"min_response_time_ms"`
	MaxResponseTime float64                `json:"max_response_time_ms"`
	P50             float64                `json:"p50_ms"`
	P90             float64                `json:"p90_ms"`
	P95             float64                `json:"p95_ms"`
	P99             float64                `json:"p99_ms"`
	OperationStats  map[string]OpStat      `json:"operation_stats"`
	TimeSeriesData  []TimeSeriesPoint      `json:"time_series_data"`
}

type OpStat struct {
	Count           int     `json:"count"`
	SuccessCount    int     `json:"success_count"`
	FailureCount    int     `json:"failure_count"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	MinResponseTime float64 `json:"min_response_time_ms"`
	MaxResponseTime float64 `json:"max_response_time_ms"`
	P95             float64 `json:"p95_ms"`
	P99             float64 `json:"p99_ms"`
}

type TimeSeriesPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	ElapsedSecs float64   `json:"elapsed_seconds"`
	OpsPerSec   float64   `json:"ops_per_second"`
	ActiveUsers int       `json:"active_users"`
}

type OperationResult struct {
	Operation    string
	Success      bool
	ResponseTime time.Duration
	Timestamp    time.Time
	ErrorMsg     string
}

var flashSaleBaseURL string

func init() {
	url, err := getFlashSaleBaseURL()
	if err != nil {
		url = os.Getenv("BASE_URL")
		if url == "" {
			fmt.Printf("Warning: Failed to get base URL: %v\n", err)
		}
	}
	flashSaleBaseURL = url
}

func getFlashSaleBaseURL() (string, error) {
	if url := os.Getenv("BASE_URL"); url != "" {
		return url, nil
	}

	terraformDir := filepath.Join("..", "terraform")
	cmd := exec.Command("terraform", "output", "-raw", "alb_url")
	cmd.Dir = terraformDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get terraform output: %v", err)
	}

	url := strings.TrimSpace(string(output))
	if url == "" {
		return "", fmt.Errorf("terraform output is empty")
	}

	return "http://" + url, nil
}

// TestFlashSale simulates a flash sale scenario with 500 concurrent users for 1 minute
func TestFlashSale(t *testing.T) {
	if flashSaleBaseURL == "" {
		t.Fatal("BASE_URL is not set. Please set BASE_URL environment variable or ensure terraform output is available")
	}

	// Check if service is healthy
	if !waitForService(t, flashSaleBaseURL, 30) {
		t.Fatal("Service did not become healthy in time")
	}

	// Test configuration
	const (
		testDuration  = 1 * time.Minute
		numUsers      = 500
		reportingInterval = 5 * time.Second
	)

	// Determine environment
	environment := "aws"
	if strings.Contains(flashSaleBaseURL, "localhost") || strings.Contains(flashSaleBaseURL, "127.0.0.1") {
		environment = "localstack"
	}

	fmt.Printf("\n╔════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║          FLASH SALE LOAD TEST - %d USERS                   ║\n", numUsers)
	fmt.Printf("╚════════════════════════════════════════════════════════════╝\n\n")
	fmt.Printf("Environment: %s\n", environment)
	fmt.Printf("Base URL: %s\n", flashSaleBaseURL)
	fmt.Printf("Duration: %v\n", testDuration)
	fmt.Printf("Concurrent Users: %d\n\n", numUsers)

	// Channels for collecting results
	resultsChan := make(chan OperationResult, numUsers*100)
	var wg sync.WaitGroup
	
	// Start time
	startTime := time.Now()
	stopChan := make(chan struct{})

	// Active user counter
	var activeUsers int64

	// Launch concurrent users
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			atomic.AddInt64(&activeUsers, 1)
			defer atomic.AddInt64(&activeUsers, -1)
			
			simulateUser(userID, resultsChan, stopChan)
		}(i)
		
		// Stagger user start times slightly to avoid thundering herd
		time.Sleep(time.Millisecond * 2)
	}

	// Monitor and collect results
	go func() {
		time.Sleep(testDuration)
		close(stopChan)
	}()

	// Progress reporting goroutine
	done := make(chan struct{})
	var results []OperationResult
	var resultsMux sync.Mutex

	go func() {
		ticker := time.NewTicker(reportingInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime)
				resultsMux.Lock()
				opsCount := len(results)
				resultsMux.Unlock()
				
				throughput := float64(opsCount) / elapsed.Seconds()
				active := atomic.LoadInt64(&activeUsers)
				
				fmt.Printf("[%6.1fs] Operations: %5d | Throughput: %7.2f ops/s | Active Users: %3d\n",
					elapsed.Seconds(), opsCount, throughput, active)
			}
		}
	}()

	// Collect results
	go func() {
		for result := range resultsChan {
			resultsMux.Lock()
			results = append(results, result)
			resultsMux.Unlock()
		}
	}()

	// Wait for all users to complete
	wg.Wait()
	close(resultsChan)
	
	// Small delay to ensure all results are collected
	time.Sleep(100 * time.Millisecond)
	close(done)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	fmt.Printf("\n✓ Test completed in %.2f seconds\n\n", duration.Seconds())

	// Analyze results
	analysisResult := analyzeFlashSaleResults(results, duration, numUsers, environment)

	// Print summary
	printFlashSaleSummary(analysisResult)

	// Save results to JSON
	filename := fmt.Sprintf("flash_sale_%s_results.json", environment)
	if err := saveFlashSaleResults(analysisResult, filename); err != nil {
		t.Errorf("Failed to save results: %v", err)
	} else {
		fmt.Printf("\n✓ Results saved to: %s\n", filename)
	}
}

// simulateUser simulates a single user's behavior during the flash sale
func simulateUser(userID int, resultsChan chan<- OperationResult, stopChan <-chan struct{}) {
	customerID := 10000 + userID
	
	for {
		select {
		case <-stopChan:
			return
		default:
			// Create cart
			cartID, createResult := performCreateCart(customerID)
			resultsChan <- createResult
			
			if !createResult.Success {
				time.Sleep(time.Millisecond * 100)
				continue
			}

			// Add random items (1-3 items)
			numItems := rand.Intn(3) + 1
			for i := 0; i < numItems; i++ {
				select {
				case <-stopChan:
					return
				default:
					productID := 1000 + rand.Intn(100) // Products 1000-1099
					quantity := rand.Intn(5) + 1       // 1-5 items
					addResult := performAddItem(cartID, productID, quantity)
					resultsChan <- addResult
					
					// Small delay between operations
					time.Sleep(time.Millisecond * time.Duration(50+rand.Intn(50)))
				}
			}

			// Get cart
			getResult := performGetCart(cartID)
			resultsChan <- getResult

			// Pause before next purchase attempt
			time.Sleep(time.Millisecond * time.Duration(100+rand.Intn(100)))
		}
	}
}

func performCreateCart(customerID int) (int64, OperationResult) {
	start := time.Now()
	
	reqBody := CreateCartRequest{CustomerID: customerID}
	jsonData, _ := json.Marshal(reqBody)
	
	resp, err := http.Post(flashSaleBaseURL+"/carts", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, OperationResult{
			Operation:    "CREATE_CART",
			Success:      false,
			ResponseTime: time.Since(start),
			Timestamp:    start,
			ErrorMsg:     err.Error(),
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusCreated {
		return 0, OperationResult{
			Operation:    "CREATE_CART",
			Success:      false,
			ResponseTime: time.Since(start),
			Timestamp:    start,
			ErrorMsg:     fmt.Sprintf("status %d: %s", resp.StatusCode, string(body)),
		}
	}

	var cartResp CreateCartResponse
	if err := json.Unmarshal(body, &cartResp); err != nil {
		return 0, OperationResult{
			Operation:    "CREATE_CART",
			Success:      false,
			ResponseTime: time.Since(start),
			Timestamp:    start,
			ErrorMsg:     err.Error(),
		}
	}

	return cartResp.ShoppingCartID, OperationResult{
		Operation:    "CREATE_CART",
		Success:      true,
		ResponseTime: time.Since(start),
		Timestamp:    start,
	}
}

func performAddItem(cartID int64, productID, quantity int) OperationResult {
	start := time.Now()
	
	reqBody := AddItemRequest{
		ProductID: productID,
		Quantity:  quantity,
	}
	jsonData, _ := json.Marshal(reqBody)
	
	url := fmt.Sprintf("%s/carts/%d/items", flashSaleBaseURL, cartID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return OperationResult{
			Operation:    "ADD_ITEM",
			Success:      false,
			ResponseTime: time.Since(start),
			Timestamp:    start,
			ErrorMsg:     err.Error(),
		}
	}
	defer resp.Body.Close()

	success := resp.StatusCode == http.StatusOK
	errMsg := ""
	if !success {
		body, _ := io.ReadAll(resp.Body)
		errMsg = fmt.Sprintf("status %d: %s", resp.StatusCode, string(body))
	}

	return OperationResult{
		Operation:    "ADD_ITEM",
		Success:      success,
		ResponseTime: time.Since(start),
		Timestamp:    start,
		ErrorMsg:     errMsg,
	}
}

func performGetCart(cartID int64) OperationResult {
	start := time.Now()
	
	url := fmt.Sprintf("%s/carts/%d", flashSaleBaseURL, cartID)
	resp, err := http.Get(url)
	if err != nil {
		return OperationResult{
			Operation:    "GET_CART",
			Success:      false,
			ResponseTime: time.Since(start),
			Timestamp:    start,
			ErrorMsg:     err.Error(),
		}
	}
	defer resp.Body.Close()

	success := resp.StatusCode == http.StatusOK
	errMsg := ""
	if !success {
		body, _ := io.ReadAll(resp.Body)
		errMsg = fmt.Sprintf("status %d: %s", resp.StatusCode, string(body))
	}

	return OperationResult{
		Operation:    "GET_CART",
		Success:      success,
		ResponseTime: time.Since(start),
		Timestamp:    start,
		ErrorMsg:     errMsg,
	}
}

func analyzeFlashSaleResults(results []OperationResult, duration time.Duration, numUsers int, environment string) FlashSaleTestResult {
	if len(results) == 0 {
		return FlashSaleTestResult{
			Environment: environment,
			TestName:    "Flash Sale Load Test",
		}
	}

	// Basic stats
	successCount := 0
	failureCount := 0
	var totalResponseTime time.Duration
	var responseTimes []float64
	opStats := make(map[string]*OpStat)

	// Initialize operation stats
	for _, op := range []string{"CREATE_CART", "ADD_ITEM", "GET_CART"} {
		opStats[op] = &OpStat{
			MinResponseTime: 999999,
		}
	}

	// Collect data
	startTime := results[0].Timestamp
	for _, result := range results {
		rtMs := float64(result.ResponseTime.Microseconds()) / 1000.0
		responseTimes = append(responseTimes, rtMs)
		totalResponseTime += result.ResponseTime

		if result.Success {
			successCount++
		} else {
			failureCount++
		}

		// Update operation stats
		stat := opStats[result.Operation]
		stat.Count++
		if result.Success {
			stat.SuccessCount++
		} else {
			stat.FailureCount++
		}

		if rtMs < stat.MinResponseTime {
			stat.MinResponseTime = rtMs
		}
		if rtMs > stat.MaxResponseTime {
			stat.MaxResponseTime = rtMs
		}
	}

	// Calculate percentiles
	sort.Float64s(responseTimes)
	p50 := percentileFlashSale(responseTimes, 50)
	p90 := percentileFlashSale(responseTimes, 90)
	p95 := percentileFlashSale(responseTimes, 95)
	p99 := percentileFlashSale(responseTimes, 99)
	minRT := responseTimes[0]
	maxRT := responseTimes[len(responseTimes)-1]

	// Calculate operation stats
	finalOpStats := make(map[string]OpStat)
	for op, stat := range opStats {
		if stat.Count > 0 {
			// Collect operation-specific response times
			var opRTs []float64
			for _, result := range results {
				if result.Operation == op {
					rtMs := float64(result.ResponseTime.Microseconds()) / 1000.0
					opRTs = append(opRTs, rtMs)
				}
			}
			sort.Float64s(opRTs)

			if len(opRTs) > 0 {
				var sum float64
				for _, rt := range opRTs {
					sum += rt
				}
				stat.AvgResponseTime = sum / float64(len(opRTs))
				stat.P95 = percentileFlashSale(opRTs, 95)
				stat.P99 = percentileFlashSale(opRTs, 99)
			}
			
			finalOpStats[op] = *stat
		}
	}

	// Generate time series data (5-second windows)
	timeSeries := generateTimeSeries(results, startTime, 5*time.Second)

	avgResponseTime := float64(totalResponseTime.Microseconds()) / float64(len(results)) / 1000.0
	successRate := float64(successCount) / float64(len(results)) * 100.0
	throughput := float64(len(results)) / duration.Seconds()

	return FlashSaleTestResult{
		Environment:     environment,
		TestName:        "Flash Sale Load Test",
		Duration:        duration.Seconds(),
		TotalUsers:      numUsers,
		TotalOperations: len(results),
		SuccessCount:    successCount,
		FailureCount:    failureCount,
		SuccessRate:     successRate,
		TotalThroughput: throughput,
		AvgResponseTime: avgResponseTime,
		MinResponseTime: minRT,
		MaxResponseTime: maxRT,
		P50:             p50,
		P90:             p90,
		P95:             p95,
		P99:             p99,
		OperationStats:  finalOpStats,
		TimeSeriesData:  timeSeries,
	}
}

func generateTimeSeries(results []OperationResult, startTime time.Time, windowSize time.Duration) []TimeSeriesPoint {
	if len(results) == 0 {
		return nil
	}

	// Find the end time
	endTime := results[0].Timestamp
	for _, r := range results {
		if r.Timestamp.After(endTime) {
			endTime = r.Timestamp
		}
	}

	duration := endTime.Sub(startTime)
	numWindows := int(duration/windowSize) + 1

	points := make([]TimeSeriesPoint, numWindows)
	windowCounts := make(map[int]int)

	// Count operations in each window
	for _, result := range results {
		elapsed := result.Timestamp.Sub(startTime)
		windowIdx := int(elapsed / windowSize)
		if windowIdx < numWindows {
			windowCounts[windowIdx]++
		}
	}

	// Create time series points
	for i := 0; i < numWindows; i++ {
		timestamp := startTime.Add(time.Duration(i) * windowSize)
		elapsedSecs := timestamp.Sub(startTime).Seconds()
		opsInWindow := windowCounts[i]
		opsPerSec := float64(opsInWindow) / windowSize.Seconds()

		points[i] = TimeSeriesPoint{
			Timestamp:   timestamp,
			ElapsedSecs: elapsedSecs,
			OpsPerSec:   opsPerSec,
			ActiveUsers: 0, // Could be calculated if we track user activity
		}
	}

	return points
}

func percentileFlashSale(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)) * float64(p) / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func printFlashSaleSummary(result FlashSaleTestResult) {
	fmt.Printf("\n╔════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                FLASH SALE TEST SUMMARY                    ║\n")
	fmt.Printf("╚════════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("Environment: %s\n", result.Environment)
	fmt.Printf("Duration: %.2f seconds\n", result.Duration)
	fmt.Printf("Total Users: %d\n", result.TotalUsers)
	fmt.Printf("Total Operations: %d\n\n", result.TotalOperations)

	fmt.Printf("╭─────────────────────────────────────╮\n")
	fmt.Printf("│          SUCCESS METRICS            │\n")
	fmt.Printf("├─────────────────────────────────────┤\n")
	fmt.Printf("│ Success Rate:  %6.2f%%            │\n", result.SuccessRate)
	fmt.Printf("│ Successful:    %6d ops          │\n", result.SuccessCount)
	fmt.Printf("│ Failed:        %6d ops          │\n", result.FailureCount)
	fmt.Printf("╰─────────────────────────────────────╯\n\n")

	fmt.Printf("╭─────────────────────────────────────╮\n")
	fmt.Printf("│       THROUGHPUT METRICS            │\n")
	fmt.Printf("├─────────────────────────────────────┤\n")
	fmt.Printf("│ Throughput:    %7.2f ops/s      │\n", result.TotalThroughput)
	fmt.Printf("│ Avg Latency:   %7.2f ms         │\n", result.AvgResponseTime)
	fmt.Printf("╰─────────────────────────────────────╯\n\n")

	fmt.Printf("╭─────────────────────────────────────╮\n")
	fmt.Printf("│        LATENCY PERCENTILES          │\n")
	fmt.Printf("├─────────────────────────────────────┤\n")
	fmt.Printf("│ Min:           %7.2f ms         │\n", result.MinResponseTime)
	fmt.Printf("│ P50 (Median):  %7.2f ms         │\n", result.P50)
	fmt.Printf("│ P90:           %7.2f ms         │\n", result.P90)
	fmt.Printf("│ P95:           %7.2f ms         │\n", result.P95)
	fmt.Printf("│ P99:           %7.2f ms         │\n", result.P99)
	fmt.Printf("│ Max:           %7.2f ms         │\n", result.MaxResponseTime)
	fmt.Printf("╰─────────────────────────────────────╯\n\n")

	fmt.Printf("╭─────────────────────────────────────────────────────────────────╮\n")
	fmt.Printf("│                    OPERATION BREAKDOWN                          │\n")
	fmt.Printf("├──────────────┬──────┬─────────┬─────────┬─────────┬────────────┤\n")
	fmt.Printf("│ Operation    │ Count│ Success │  Avg ms │  P95 ms │   P99 ms   │\n")
	fmt.Printf("├──────────────┼──────┼─────────┼─────────┼─────────┼────────────┤\n")

	for _, op := range []string{"CREATE_CART", "ADD_ITEM", "GET_CART"} {
		if stat, ok := result.OperationStats[op]; ok {
			successRate := float64(stat.SuccessCount) / float64(stat.Count) * 100.0
			fmt.Printf("│ %-12s │ %4d │ %6.2f%% │ %7.2f │ %7.2f │ %10.2f │\n",
				op, stat.Count, successRate, stat.AvgResponseTime, stat.P95, stat.P99)
		}
	}
	fmt.Printf("╰──────────────┴──────┴─────────┴─────────┴─────────┴────────────╯\n")
}

func saveFlashSaleResults(result FlashSaleTestResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func waitForService(t *testing.T, baseURL string, timeoutSecs int) bool {
	fmt.Printf("Checking service health...\n")
	
	for i := 0; i < timeoutSecs; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("✓ Service is healthy\n\n")
				return true
			}
		}
		time.Sleep(time.Second)
	}
	
	fmt.Printf("✗ Service did not become healthy\n")
	return false
}
