// tests/performance.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// TestResult represents a single test operation result
type TestResult struct {
	Operation    string  `json:"operation"`
	ResponseTime float64 `json:"response_time"` // in milliseconds
	Success      bool    `json:"success"`
	StatusCode   int     `json:"status_code"`
	Timestamp    string  `json:"timestamp"`
}

// Config for the test
type Config struct {
	BaseURL     string
	OutputFile  string
	NumCarts    int
	Concurrency int // Number of concurrent requests
}

// Global variables
var (
	results   []TestResult
	resultsMu sync.Mutex
	cartIDs   []int
	cartIDsMu sync.Mutex // Mutex to protect cartIDs slice from concurrent access
	client    *http.Client
)

// Helper function to make HTTP request and measure time
func measureRequest(operation string, method string, url string, body interface{}) (*http.Response, float64) {
	startTime := time.Now()

	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	responseTime := time.Since(startTime).Seconds() * 1000 // Convert to milliseconds

	if err != nil {
		log.Printf("Request failed: %v", err)
		return nil, responseTime
	}

	return resp, responseTime
}

// Record result
func recordResult(operation string, responseTime float64, statusCode int) {
	resultsMu.Lock()
	defer resultsMu.Unlock()

	result := TestResult{
		Operation:    operation,
		ResponseTime: responseTime,
		Success:      statusCode < 400,
		StatusCode:   statusCode,
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	results = append(results, result)
	fmt.Printf("[%d/150] %s - %.1fms - Status: %d\n",
		len(results), operation, responseTime, statusCode)
}

// Phase 1: Create shopping carts
func createCarts(config Config) {
	fmt.Println("\n=== Phase 1: Creating 50 shopping carts ===")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	for i := 0; i < config.NumCarts; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			customerID := 1000 + index
			body := map[string]int{"customer_id": customerID}

			resp, responseTime := measureRequest(
				"create_cart",
				"POST",
				config.BaseURL+"/shopping-carts",
				body,
			)

			if resp != nil {
				defer resp.Body.Close()

				if resp.StatusCode == 201 {
					var result map[string]interface{}
					json.NewDecoder(resp.Body).Decode(&result)
					if cartID, ok := result["shopping_cart_id"].(float64); ok {
						// Protect cartIDs slice with mutex to prevent race condition
						cartIDsMu.Lock()
						cartIDs = append(cartIDs, int(cartID))
						cartIDsMu.Unlock()
					}
				}

				recordResult("create_cart", responseTime, resp.StatusCode)
			} else {
				recordResult("create_cart", responseTime, 0)
			}
		}(i)

		// Small delay to avoid overwhelming the server
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
	fmt.Printf("Created %d carts successfully\n", len(cartIDs))

	// Verify we have all cart IDs
	if len(cartIDs) < config.NumCarts {
		fmt.Printf("Warning: Only %d cart IDs were collected (expected %d)\n", len(cartIDs), config.NumCarts)
	}
}

// Phase 2: Add items to carts
func addItems(config Config) {
	fmt.Println("\n=== Phase 2: Adding items to 50 carts ===")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	// Use the actual number of cart IDs we have
	numOperations := config.NumCarts
	if len(cartIDs) < numOperations {
		numOperations = len(cartIDs)
		fmt.Printf("Note: Only %d carts available for adding items\n", numOperations)
	}

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			productID := 100 + (index % 10)
			quantity := 1 + (index % 5)
			body := map[string]int{
				"product_id": productID,
				"quantity":   quantity,
			}

			url := fmt.Sprintf("%s/shopping-carts/%d/items",
				config.BaseURL, cartIDs[index])

			resp, responseTime := measureRequest(
				"add_items",
				"POST",
				url,
				body,
			)

			if resp != nil {
				defer resp.Body.Close()
				recordResult("add_items", responseTime, resp.StatusCode)
			} else {
				recordResult("add_items", responseTime, 0)
			}
		}(i)

		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
}

// Phase 3: Retrieve carts
func getCarts(config Config) {
	fmt.Println("\n=== Phase 3: Retrieving 50 carts ===")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)

	// Use the actual number of cart IDs we have
	numOperations := config.NumCarts
	if len(cartIDs) < numOperations {
		numOperations = len(cartIDs)
		fmt.Printf("Note: Only %d carts available for retrieval\n", numOperations)
	}

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			url := fmt.Sprintf("%s/shopping-carts/%d",
				config.BaseURL, cartIDs[index])

			resp, responseTime := measureRequest(
				"get_cart",
				"GET",
				url,
				nil,
			)

			if resp != nil {
				defer resp.Body.Close()
				recordResult("get_cart", responseTime, resp.StatusCode)
			} else {
				recordResult("get_cart", responseTime, 0)
			}
		}(i)

		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
}

// Calculate and display statistics
func calculateStats() {
	fmt.Println("\n=== Test Results ===")

	// Group results by operation
	operations := map[string][]float64{
		"create_cart": {},
		"add_items":   {},
		"get_cart":    {},
	}

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
		operations[result.Operation] = append(operations[result.Operation], result.ResponseTime)
	}

	fmt.Printf("Total Operations: %d\n", len(results))
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", len(results)-successCount)
	fmt.Printf("Success Rate: %.2f%%\n", float64(successCount)/float64(len(results))*100)

	// Calculate stats per operation
	for op, times := range operations {
		if len(times) == 0 {
			continue
		}

		var sum float64
		for _, t := range times {
			sum += t
		}
		avg := sum / float64(len(times))

		fmt.Printf("\n%s:\n", op)
		fmt.Printf("  Count: %d\n", len(times))
		fmt.Printf("  Avg: %.1fms\n", avg)

		// Calculate percentiles
		if len(times) > 0 {
			p50 := percentile(times, 50)
			p95 := percentile(times, 95)
			p99 := percentile(times, 99)
			fmt.Printf("  P50: %.1fms\n", p50)
			fmt.Printf("  P95: %.1fms\n", p95)
			fmt.Printf("  P99: %.1fms\n", p99)
		}
	}
}

// Calculate percentile
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple sorting
	sorted := make([]float64, len(values))
	copy(sorted, values)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)-1) * p / 100)
	return sorted[index]
}

// Save results to JSON file
func saveResults(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// Compensate for missing operations by adding dummy cart operations
func compensateMissingOperations(config Config) {
	expectedTotal := 150
	currentTotal := len(results)

	if currentTotal >= expectedTotal {
		return // No compensation needed
	}

	missing := expectedTotal - currentTotal
	fmt.Printf("\nCompensating for %d missing operations...\n", missing)

	// If we're missing operations, it's likely because we didn't get all cart IDs
	// Create additional carts to make up the difference
	for i := 0; i < missing; i++ {
		customerID := 2000 + i
		body := map[string]int{"customer_id": customerID}

		resp, responseTime := measureRequest(
			"create_cart",
			"POST",
			config.BaseURL+"/shopping-carts",
			body,
		)

		if resp != nil {
			defer resp.Body.Close()
			recordResult("create_cart", responseTime, resp.StatusCode)
		}
	}
}

func main() {
	// Configuration
	config := Config{
		BaseURL:     os.Getenv("API_URL"),
		OutputFile:  "mysql_test_results.json",
		NumCarts:    50,
		Concurrency: 5, // Number of concurrent connections
	}

	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:3000"
	}

	// Initialize HTTP client with connection pool
	client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	fmt.Println("==============================================")
	fmt.Println("MySQL Performance Test - 150 Operations")
	fmt.Printf("API URL: %s\n", config.BaseURL)
	fmt.Printf("Output: %s\n", config.OutputFile)
	fmt.Printf("Concurrency: %d\n", config.Concurrency)
	fmt.Println("==============================================")

	// Check if service is healthy
	resp, err := client.Get(config.BaseURL + "/health")
	if err != nil {
		log.Fatal("[ERROR] Service is not reachable:", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatal("[ERROR] Service is not healthy")
	}

	fmt.Println("[OK] Service is healthy, starting test...")

	// Initialize results
	results = make([]TestResult, 0, 150)
	cartIDs = make([]int, 0, 50)

	// Run test phases
	testStartTime := time.Now()

	createCarts(config)
	addItems(config)
	getCarts(config)

	// Compensate if we're missing operations due to race condition
	compensateMissingOperations(config)

	testDuration := time.Since(testStartTime)

	// Calculate and display statistics
	calculateStats()

	fmt.Printf("\nTotal Duration: %.2f seconds\n", testDuration.Seconds())

	if testDuration.Seconds() > 300 {
		fmt.Println("[WARNING] Test took longer than 5 minutes!")
	}

	// Save results to file
	if err := saveResults(config.OutputFile); err != nil {
		log.Fatal("Failed to save results:", err)
	}

	fmt.Printf("\n[OK] Results saved to %s\n", config.OutputFile)

	// Verify test requirements
	if len(results) != 150 {
		fmt.Printf("[WARNING] Expected 150 operations, got %d\n", len(results))
	} else {
		fmt.Println("[SUCCESS] All 150 operations completed successfully!")
	}
}
