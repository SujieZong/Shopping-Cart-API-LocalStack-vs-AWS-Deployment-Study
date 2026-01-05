package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var loadTestBaseURL string

// ScalingTestResult stores results for different load levels
type ScalingTestResult struct {
	LoadLevel       int                    `json:"load_level"` // 150, 300, 500
	TotalOperations int                    `json:"total_operations"`
	Duration        float64                `json:"duration_seconds"`
	Throughput      float64                `json:"throughput_ops_per_sec"`
	SuccessRate     float64                `json:"success_rate_percent"`
	AvgResponseTime float64                `json:"avg_response_time_ms"`
	P50             float64                `json:"p50_ms"`
	P95             float64                `json:"p95_ms"`
	P99             float64                `json:"p99_ms"`
	Results         []TestResult           `json:"results"`
	OperationStats  map[string]OpStats     `json:"operation_stats"`
}

type OpStats struct {
	Count           int     `json:"count"`
	SuccessCount    int     `json:"success_count"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	P50             float64 `json:"p50_ms"`
	P95             float64 `json:"p95_ms"`
	P99             float64 `json:"p99_ms"`
}

func init() {
	url, err := getLoadTestBaseURL()
	if err != nil {
		// Fallback to environment variable if terraform output fails
		url = os.Getenv("BASE_URL")
		if url == "" {
			fmt.Printf("Warning: Failed to get base URL from Terraform: %v\n", err)
			fmt.Println("You can set BASE_URL environment variable as a fallback")
			fmt.Println("Example: export BASE_URL=http://your-alb-url.amazonaws.com")
			os.Exit(1)
		}
		fmt.Printf("Using BASE_URL from environment variable: %s\n", url)
	} else {
		fmt.Printf("Successfully retrieved base URL from Terraform: %s\n", url)
	}
	loadTestBaseURL = url
}

func getLoadTestBaseURL() (string, error) {
	terraformDir := filepath.Join("..", "terraform")
	cmd := exec.Command("terraform", "output", "-raw", "alb_url")
	cmd.Dir = terraformDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get terraform output: %v\nOutput: %s", err, string(output))
	}
	
	url := strings.TrimSpace(string(output))
	
	// Check if the output contains terraform warnings or is invalid
	if url == "" || strings.Contains(url, "Warning:") || strings.Contains(url, "╷") || strings.Contains(url, "\x1b[") {
		return "", fmt.Errorf("terraform output returned invalid or empty URL")
	}
	
	return url, nil
}

// runLoadTest executes a load test with specified number of operations
func runLoadTest(numCarts int, client *http.Client) ScalingTestResult {
	var allResults []TestResult
	var cartIDs []int64
	var mu sync.Mutex
	
	startTime := time.Now()
	
	fmt.Printf("\n=== Running load test with %d carts ===\n", numCarts)
	
	// Phase 1: Create carts
	fmt.Printf("Creating %d carts...\n", numCarts)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20) // Control concurrency
	
	for i := 0; i < numCarts; i++ {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(idx int) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			customerID := 60000 + idx
			result, cartID := createLoadTestCart(client, customerID)
			
			mu.Lock()
			allResults = append(allResults, result)
			if cartID > 0 {
				cartIDs = append(cartIDs, cartID)
			}
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	time.Sleep(500 * time.Millisecond)
	
	fmt.Printf("Created %d carts\n", len(cartIDs))
	
	// Phase 2: Add items
	fmt.Printf("Adding items to %d carts...\n", numCarts)
	for i := 0; i < numCarts && i < len(cartIDs); i++ {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(idx int) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			cartID := cartIDs[idx]
			productID := 2000 + idx
			quantity := (idx % 10) + 1
			
			result := addLoadTestItem(client, cartID, productID, quantity)
			
			mu.Lock()
			allResults = append(allResults, result)
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	time.Sleep(500 * time.Millisecond)
	
	// Phase 3: Get carts
	fmt.Printf("Retrieving %d carts...\n", numCarts)
	for i := 0; i < numCarts && i < len(cartIDs); i++ {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(idx int) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			cartID := cartIDs[idx]
			result := getLoadTestCart(client, cartID)
			
			mu.Lock()
			allResults = append(allResults, result)
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	
	duration := time.Since(startTime).Seconds()
	
	// Calculate statistics
	return calculateScalingStats(allResults, numCarts, duration)
}

func createLoadTestCart(client *http.Client, customerID int) (TestResult, int64) {
	start := time.Now()
	timestamp := start.Format(time.RFC3339)
	
	req := CreateCartRequest{CustomerID: customerID}
	body, _ := json.Marshal(req)
	
	resp, err := client.Post(
		loadTestBaseURL+"/shopping-carts",
		"application/json",
		bytes.NewBuffer(body),
	)
	
	responseTime := float64(time.Since(start).Microseconds()) / 1000.0
	
	result := TestResult{
		Operation:    "create_cart",
		ResponseTime: responseTime,
		Success:      false,
		StatusCode:   0,
		Timestamp:    timestamp,
	}
	
	var cartID int64
	
	if err != nil {
		fmt.Printf("[ERROR] Failed to create cart (customer_id=%d): %v\n", customerID, err)
		return result, 0
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode == http.StatusCreated
	
	if result.Success {
		var cartResp CreateCartResponse
		if err := json.NewDecoder(resp.Body).Decode(&cartResp); err == nil {
			cartID = cartResp.ShoppingCartID
		} else {
			fmt.Printf("[ERROR] Failed to decode cart response: %v\n", err)
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[ERROR] Create cart failed with status %d: %s\n", resp.StatusCode, string(body))
	}
	
	return result, cartID
}

func addLoadTestItem(client *http.Client, cartID int64, productID, quantity int) TestResult {
	start := time.Now()
	timestamp := start.Format(time.RFC3339)
	
	req := AddItemRequest{ProductID: productID, Quantity: quantity}
	body, _ := json.Marshal(req)
	
	url := fmt.Sprintf("%s/shopping-carts/%d/items", loadTestBaseURL, cartID)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	
	responseTime := float64(time.Since(start).Microseconds()) / 1000.0
	
	result := TestResult{
		Operation:    "add_items",
		ResponseTime: responseTime,
		Success:      false,
		StatusCode:   0,
		Timestamp:    timestamp,
	}
	
	if err != nil {
		fmt.Printf("[ERROR] Failed to add item (cart_id=%d): %v\n", cartID, err)
		return result
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode == http.StatusNoContent
	if !result.Success {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[ERROR] Add item failed with status %d: %s\n", resp.StatusCode, string(body))
	} else {
		io.ReadAll(resp.Body)
	}
	
	return result
}

func getLoadTestCart(client *http.Client, cartID int64) TestResult {
	start := time.Now()
	timestamp := start.Format(time.RFC3339)
	
	url := fmt.Sprintf("%s/shopping-carts/%d", loadTestBaseURL, cartID)
	resp, err := client.Get(url)
	
	responseTime := float64(time.Since(start).Microseconds()) / 1000.0
	
	result := TestResult{
		Operation:    "get_cart",
		ResponseTime: responseTime,
		Success:      false,
		StatusCode:   0,
		Timestamp:    timestamp,
	}
	
	if err != nil {
		fmt.Printf("[ERROR] Failed to get cart (cart_id=%d): %v\n", cartID, err)
		return result
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode == http.StatusOK
	if !result.Success {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[ERROR] Get cart failed with status %d: %s\n", resp.StatusCode, string(body))
	} else {
		io.ReadAll(resp.Body)
	}
	
	return result
}

func calculateScalingStats(results []TestResult, loadLevel int, duration float64) ScalingTestResult {
	totalOps := len(results)
	successCount := 0
	var totalResponseTime float64
	
	opMap := make(map[string][]float64)
	
	for _, r := range results {
		if r.Success {
			successCount++
		}
		totalResponseTime += r.ResponseTime
		opMap[r.Operation] = append(opMap[r.Operation], r.ResponseTime)
	}
	
	avgResponseTime := 0.0
	if totalOps > 0 {
		avgResponseTime = totalResponseTime / float64(totalOps)
	}
	
	allTimes := make([]float64, 0, totalOps)
	for _, r := range results {
		allTimes = append(allTimes, r.ResponseTime)
	}
	
	// Calculate operation-specific stats
	opStats := make(map[string]OpStats)
	for op, times := range opMap {
		successOps := 0
		for _, r := range results {
			if r.Operation == op && r.Success {
				successOps++
			}
		}
		
		var sum float64
		for _, t := range times {
			sum += t
		}
		
		opStats[op] = OpStats{
			Count:           len(times),
			SuccessCount:    successOps,
			AvgResponseTime: sum / float64(len(times)),
			P50:             percentile(times, 50),
			P95:             percentile(times, 95),
			P99:             percentile(times, 99),
		}
	}
	
	throughput := 0.0
	if duration > 0 {
		throughput = float64(totalOps) / duration
	}
	
	successRate := 0.0
	if totalOps > 0 {
		successRate = (float64(successCount) / float64(totalOps)) * 100
	}
	
	return ScalingTestResult{
		LoadLevel:       loadLevel,
		TotalOperations: totalOps,
		Duration:        duration,
		Throughput:      throughput,
		SuccessRate:     successRate,
		AvgResponseTime: avgResponseTime,
		P50:             percentile(allTimes, 50),
		P95:             percentile(allTimes, 95),
		P99:             percentile(allTimes, 99),
		Results:         results,
		OperationStats:  opStats,
	}
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	// Simple bubble sort
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

func TestLoadScaling(t *testing.T) {
	// Check server connectivity before running tests
	fmt.Printf("\n=== Checking server connectivity at %s ===\n", loadTestBaseURL)
	testClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := testClient.Get(loadTestBaseURL + "/health")
	if err != nil {
		// Try a simple request to root
		resp, err = testClient.Get(loadTestBaseURL)
	}
	if err != nil {
		t.Fatalf("❌ Cannot connect to server at %s: %v\n\nPlease ensure the API is running. You can:\n1. Start it locally: cd ../shopping-cart-api && ./run-local.sh\n2. Or set BASE_URL to a deployed instance", loadTestBaseURL, err)
	}
	if resp != nil {
		resp.Body.Close()
		fmt.Printf("✓ Server is reachable (status: %d)\n", resp.StatusCode)
	}
	
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
		Timeout: 30 * time.Second,
	}
	
	loadLevels := []int{50, 100, 167} // 150, 300, 500 total ops
	var allResults []ScalingTestResult
	
	for _, numCarts := range loadLevels {
		fmt.Printf("\n\n╔════════════════════════════════════════╗\n")
		fmt.Printf("║  Testing Load Level: %d operations     ║\n", numCarts*3)
		fmt.Printf("╚════════════════════════════════════════╝\n")
		
		result := runLoadTest(numCarts, client)
		allResults = append(allResults, result)
		
		// Print summary
		fmt.Printf("\n--- Results for %d operations ---\n", numCarts*3)
		fmt.Printf("Duration: %.2f seconds\n", result.Duration)
		fmt.Printf("Throughput: %.2f ops/sec\n", result.Throughput)
		fmt.Printf("Success Rate: %.2f%%\n", result.SuccessRate)
		fmt.Printf("Avg Response Time: %.2f ms\n", result.AvgResponseTime)
		fmt.Printf("P50: %.2f ms\n", result.P50)
		fmt.Printf("P95: %.2f ms\n", result.P95)
		fmt.Printf("P99: %.2f ms\n", result.P99)
		
		// Wait between load levels
		if numCarts != loadLevels[len(loadLevels)-1] {
			fmt.Println("\nWaiting 10 seconds before next load level...")
			time.Sleep(10 * time.Second)
		}
	}
	
	// Save results
	outputFile := "load_scaling_results.json"
	file, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(allResults); err != nil {
		t.Fatalf("Failed to write JSON: %v", err)
	}
	
	fmt.Printf("\n\n✓ Results saved to: %s\n", outputFile)
	
	// Check if any tests had acceptable success rates
	hasFailures := false
	for _, r := range allResults {
		if r.SuccessRate < 50.0 {
			hasFailures = true
			break
		}
	}
	
	// Print comparison summary
	fmt.Println("\n\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              LOAD SCALING COMPARISON SUMMARY                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n%-15s %-15s %-15s %-15s %-15s\n", "Load Level", "Throughput", "Avg Latency", "P95", "Success Rate")
	fmt.Println(strings.Repeat("-", 80))
	
	for _, r := range allResults {
		fmt.Printf("%-15d %-15.2f %-15.2f %-15.2f %-15.2f%%\n",
			r.LoadLevel,
			r.Throughput,
			r.AvgResponseTime,
			r.P95,
			r.SuccessRate)
	}
	
	if hasFailures {
		t.Fatalf("\n❌ Test failed: One or more load levels had success rate below 50%%")
	}
}
