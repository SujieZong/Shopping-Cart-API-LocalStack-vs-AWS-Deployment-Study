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
	"testing"
	"time"
)

// TestResult represents a single operation result
type TestResult struct {
	Operation    string  `json:"operation"`
	ResponseTime float64 `json:"response_time"` // in milliseconds
	Success      bool    `json:"success"`
	StatusCode   int     `json:"status_code"`
	Timestamp    string  `json:"timestamp"`
}

var performanceBaseURL string

// Create a shared HTTP client with connection pooling for better performance
var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	},
	Timeout: 30 * time.Second,
}

// getPerformanceBaseURL retrieves the ALB URL from Terraform output
func getPerformanceBaseURL() (string, error) {
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

// init function to set baseURL from Terraform output
func init() {
	url, err := getPerformanceBaseURL()
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
	performanceBaseURL = url
}

// createCartOperation performs a POST /shopping-carts operation and returns cart ID
func createCartOperation(customerID int) (TestResult, int64) {
	start := time.Now()
	timestamp := start.Format(time.RFC3339)
	
	req := CreateCartRequest{CustomerID: customerID}
	body, _ := json.Marshal(req)
	
	resp, err := httpClient.Post(
		performanceBaseURL+"/shopping-carts",
		"application/json",
		bytes.NewBuffer(body),
	)
	
	responseTime := float64(time.Since(start).Microseconds()) / 1000.0 // Convert to milliseconds
	
	result := TestResult{
		Operation:    "create_cart",
		ResponseTime: responseTime,
		Success:      false,
		StatusCode:   0,
		Timestamp:    timestamp,
	}
	
	var cartID int64
	
	if err != nil {
		fmt.Printf("Error creating cart: %v\n", err)
		return result, 0
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode == http.StatusCreated
	
	// Extract cart ID from response
	if result.Success {
		var cartResp CreateCartResponse
		if err := json.NewDecoder(resp.Body).Decode(&cartResp); err == nil {
			cartID = cartResp.ShoppingCartID
		}
	} else {
		io.ReadAll(resp.Body) // Drain body for connection reuse
	}
	
	return result, cartID
}

// addItemOperation performs a POST /shopping-carts/{id}/items operation
func addItemOperation(cartID int64, productID, quantity int) TestResult {
	start := time.Now()
	timestamp := start.Format(time.RFC3339)
	
	req := AddItemRequest{
		ProductID: productID,
		Quantity:  quantity,
	}
	body, _ := json.Marshal(req)
	
	url := fmt.Sprintf("%s/shopping-carts/%d/items", performanceBaseURL, cartID)
	resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	
	responseTime := float64(time.Since(start).Microseconds()) / 1000.0 // Convert to milliseconds
	
	result := TestResult{
		Operation:    "add_items",
		ResponseTime: responseTime,
		Success:      false,
		StatusCode:   0,
		Timestamp:    timestamp,
	}
	
	if err != nil {
		fmt.Printf("Error adding item: %v\n", err)
		return result
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode == http.StatusNoContent
	
	// Drain response body for connection reuse
	io.ReadAll(resp.Body)
	
	return result
}

// getCartOperation performs a GET /shopping-carts/{id} operation
func getCartOperation(cartID int64) TestResult {
	start := time.Now()
	timestamp := start.Format(time.RFC3339)
	
	url := fmt.Sprintf("%s/shopping-carts/%d", performanceBaseURL, cartID)
	resp, err := httpClient.Get(url)
	
	responseTime := float64(time.Since(start).Microseconds()) / 1000.0 // Convert to milliseconds
	
	result := TestResult{
		Operation:    "get_cart",
		ResponseTime: responseTime,
		Success:      false,
		StatusCode:   0,
		Timestamp:    timestamp,
	}
	
	if err != nil {
		fmt.Printf("Error getting cart: %v\n", err)
		return result
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode == http.StatusOK
	
	// Drain response body for connection reuse
	io.ReadAll(resp.Body)
	
	return result
}

// TestDynamoDBPerformance runs 50 create, 50 add, 50 get operations
func TestDynamoDBPerformance(t *testing.T) {
	var allResults []TestResult
	var cartIDs []int64
	
	fmt.Println("\n=== Starting DynamoDB Performance Test ===")
	fmt.Println("Running 50 CREATE operations...")
	
	// Phase 1: Create 50 shopping carts
	for i := 0; i < 50; i++ {
		customerID := 50000 + i
		result, cartID := createCartOperation(customerID)
		allResults = append(allResults, result)
		
		if cartID > 0 {
			cartIDs = append(cartIDs, cartID)
		}
		
		// Small delay to avoid overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}
	
	fmt.Printf("Completed CREATE operations. Created %d carts.\n", len(cartIDs))
	
	if len(cartIDs) == 0 {
		t.Fatal("Failed to create any carts. Cannot continue test.")
	}
	
	// Wait a bit for cart creation to propagate
	time.Sleep(500 * time.Millisecond)
	
	// Phase 2: Add items to 50 carts
	fmt.Println("Running 50 ADD_ITEMS operations...")
	for i := 0; i < 50 && i < len(cartIDs); i++ {
		cartID := cartIDs[i]
		productID := 1000 + i
		quantity := (i % 10) + 1 // Quantities from 1 to 10
		
		result := addItemOperation(cartID, productID, quantity)
		allResults = append(allResults, result)
		
		time.Sleep(10 * time.Millisecond)
	}
	
	fmt.Println("Completed ADD_ITEMS operations.")
	
	// Wait a bit for items to propagate
	time.Sleep(500 * time.Millisecond)
	
	// Phase 3: Get 50 carts
	fmt.Println("Running 50 GET_CART operations...")
	for i := 0; i < 50 && i < len(cartIDs); i++ {
		cartID := cartIDs[i]
		
		result := getCartOperation(cartID)
		allResults = append(allResults, result)
		
		time.Sleep(10 * time.Millisecond)
	}
	
	fmt.Println("Completed GET_CART operations.")
	
	// Save results to JSON file
	outputFile := "dynamodb_test_results.json"
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
	
	fmt.Printf("\nResults saved to: %s\n", outputFile)
	
	// Print summary statistics
	printSummary(allResults)
}

// printSummary calculates and prints summary statistics
func printSummary(results []TestResult) {
	fmt.Println("\n=== Performance Test Summary ===")
	
	// Count operations by type
	operationStats := make(map[string]struct {
		total       int
		successful  int
		totalTime   float64
		minTime     float64
		maxTime     float64
	})
	
	for _, result := range results {
		stats := operationStats[result.Operation]
		stats.total++
		if result.Success {
			stats.successful++
		}
		stats.totalTime += result.ResponseTime
		
		if stats.minTime == 0 || result.ResponseTime < stats.minTime {
			stats.minTime = result.ResponseTime
		}
		if result.ResponseTime > stats.maxTime {
			stats.maxTime = result.ResponseTime
		}
		
		operationStats[result.Operation] = stats
	}
	
	// Print statistics for each operation type
	for operation, stats := range operationStats {
		fmt.Printf("\n%s:\n", strings.ToUpper(operation))
		fmt.Printf("  Total: %d\n", stats.total)
		fmt.Printf("  Successful: %d (%.2f%%)\n", 
			stats.successful, 
			float64(stats.successful)/float64(stats.total)*100)
		fmt.Printf("  Failed: %d\n", stats.total-stats.successful)
		
		if stats.total > 0 {
			avgTime := stats.totalTime / float64(stats.total)
			fmt.Printf("  Response Time:\n")
			fmt.Printf("    Average: %.2f ms\n", avgTime)
			fmt.Printf("    Min: %.2f ms\n", stats.minTime)
			fmt.Printf("    Max: %.2f ms\n", stats.maxTime)
		}
	}
	
	// Overall statistics
	totalOps := len(results)
	successfulOps := 0
	totalTime := 0.0
	
	for _, result := range results {
		if result.Success {
			successfulOps++
		}
		totalTime += result.ResponseTime
	}
	
	fmt.Printf("\nOVERALL:\n")
	fmt.Printf("  Total Operations: %d\n", totalOps)
	fmt.Printf("  Successful: %d (%.2f%%)\n", 
		successfulOps, 
		float64(successfulOps)/float64(totalOps)*100)
	fmt.Printf("  Failed: %d\n", totalOps-successfulOps)
	fmt.Printf("  Average Response Time: %.2f ms\n", totalTime/float64(totalOps))
	
	fmt.Println("\n================================")
}
