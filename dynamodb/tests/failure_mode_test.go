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

var failureTestBaseURL string

// FailureTestResult stores results for failure mode testing
type FailureTestResult struct {
	TestName        string  `json:"test_name"`
	Description     string  `json:"description"`
	RequestSent     string  `json:"request_sent"`
	ExpectedStatus  int     `json:"expected_status"`
	ActualStatus    int     `json:"actual_status"`
	ResponseTime    float64 `json:"response_time_ms"`
	Success         bool    `json:"success"` // Did it fail as expected?
	ErrorMessage    string  `json:"error_message,omitempty"`
	ResponseBody    string  `json:"response_body,omitempty"`
	Timestamp       string  `json:"timestamp"`
}

func init() {
	url, err := getFailureTestBaseURL()
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
	failureTestBaseURL = url
}

func getFailureTestBaseURL() (string, error) {
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

// sendFailureTestRequest sends a request and measures the response
func sendFailureTestRequest(method, url string, body interface{}, expectedStatus int) FailureTestResult {
	start := time.Now()
	timestamp := start.Format(time.RFC3339)
	
	var reqBody io.Reader
	bodyStr := ""
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
		bodyStr = string(jsonBody)
	}
	
	req, _ := http.NewRequest(method, url, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	responseTime := float64(time.Since(start).Microseconds()) / 1000.0
	
	result := FailureTestResult{
		RequestSent:    bodyStr,
		ExpectedStatus: expectedStatus,
		ResponseTime:   responseTime,
		Timestamp:      timestamp,
	}
	
	if err != nil {
		result.ErrorMessage = err.Error()
		result.Success = false
		return result
	}
	defer resp.Body.Close()
	
	result.ActualStatus = resp.StatusCode
	result.Success = (resp.StatusCode == expectedStatus)
	
	bodyBytes, _ := io.ReadAll(resp.Body)
	result.ResponseBody = string(bodyBytes)
	
	return result
}

func TestInvalidRequests(t *testing.T) {
	fmt.Println("\n╔════════════════════════════════════════╗")
	fmt.Println("║      FAILURE MODE: INVALID REQUESTS    ║")
	fmt.Println("╚════════════════════════════════════════╝")
	
	var allResults []FailureTestResult
	
	// Test 1: Create cart with missing customer_id
	t.Run("MissingCustomerID", func(t *testing.T) {
		result := sendFailureTestRequest(
			"POST",
			failureTestBaseURL+"/shopping-carts",
			map[string]interface{}{}, // Empty body
			http.StatusBadRequest,
		)
		result.TestName = "missing_customer_id"
		result.Description = "Create cart without customer_id field"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		if !result.Success {
			t.Errorf("API did not handle missing customer_id correctly")
		}
		
		allResults = append(allResults, result)
	})
	
	// Test 2: Create cart with invalid customer_id (string instead of int)
	t.Run("InvalidCustomerIDType", func(t *testing.T) {
		result := sendFailureTestRequest(
			"POST",
			failureTestBaseURL+"/shopping-carts",
			map[string]interface{}{"customer_id": "not_a_number"},
			http.StatusBadRequest,
		)
		result.TestName = "invalid_customer_id_type"
		result.Description = "Create cart with string customer_id instead of integer"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Test 3: Create cart with negative customer_id
	t.Run("NegativeCustomerID", func(t *testing.T) {
		result := sendFailureTestRequest(
			"POST",
			failureTestBaseURL+"/shopping-carts",
			map[string]interface{}{"customer_id": -999},
			http.StatusBadRequest,
		)
		result.TestName = "negative_customer_id"
		result.Description = "Create cart with negative customer_id"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Test 4: Add item with missing product_id
	t.Run("MissingProductID", func(t *testing.T) {
		// First create a valid cart
		validCart := CreateCartRequest{CustomerID: 88888}
		body, _ := json.Marshal(validCart)
		resp, _ := http.Post(failureTestBaseURL+"/shopping-carts", "application/json", bytes.NewBuffer(body))
		var cartResp CreateCartResponse
		json.NewDecoder(resp.Body).Decode(&cartResp)
		resp.Body.Close()
		
		// Now try to add item without product_id
		result := sendFailureTestRequest(
			"POST",
			fmt.Sprintf("%s/shopping-carts/%d/items", failureTestBaseURL, cartResp.ShoppingCartID),
			map[string]interface{}{"quantity": 5},
			http.StatusBadRequest,
		)
		result.TestName = "missing_product_id"
		result.Description = "Add item without product_id field"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Test 5: Add item with zero/negative quantity
	t.Run("InvalidQuantity", func(t *testing.T) {
		// Create a valid cart
		validCart := CreateCartRequest{CustomerID: 88889}
		body, _ := json.Marshal(validCart)
		resp, _ := http.Post(failureTestBaseURL+"/shopping-carts", "application/json", bytes.NewBuffer(body))
		var cartResp CreateCartResponse
		json.NewDecoder(resp.Body).Decode(&cartResp)
		resp.Body.Close()
		
		// Try adding with zero quantity
		result := sendFailureTestRequest(
			"POST",
			fmt.Sprintf("%s/shopping-carts/%d/items", failureTestBaseURL, cartResp.ShoppingCartID),
			map[string]interface{}{"product_id": 100, "quantity": 0},
			http.StatusBadRequest,
		)
		result.TestName = "zero_quantity"
		result.Description = "Add item with zero quantity"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Test 6: Malformed JSON
	t.Run("MalformedJSON", func(t *testing.T) {
		start := time.Now()
		req, _ := http.NewRequest("POST", failureTestBaseURL+"/shopping-carts", 
			bytes.NewBuffer([]byte("{invalid json")))
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, _ := client.Do(req)
		responseTime := float64(time.Since(start).Microseconds()) / 1000.0
		
		result := FailureTestResult{
			TestName:       "malformed_json",
			Description:    "Send malformed JSON in request body",
			RequestSent:    "{invalid json",
			ExpectedStatus: http.StatusBadRequest,
			ActualStatus:   resp.StatusCode,
			ResponseTime:   responseTime,
			Success:        resp.StatusCode == http.StatusBadRequest,
			Timestamp:      time.Now().Format(time.RFC3339),
		}
		resp.Body.Close()
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Save results
	saveFailureResults(t, allResults, "invalid_requests")
}

func TestMissingResources(t *testing.T) {
	fmt.Println("\n╔════════════════════════════════════════╗")
	fmt.Println("║    FAILURE MODE: MISSING RESOURCES     ║")
	fmt.Println("╚════════════════════════════════════════╝")
	
	var allResults []FailureTestResult
	
	// Test 1: Get non-existent cart
	t.Run("NonExistentCart", func(t *testing.T) {
		result := sendFailureTestRequest(
			"GET",
			fmt.Sprintf("%s/shopping-carts/999999999", failureTestBaseURL),
			nil,
			http.StatusNotFound,
		)
		result.TestName = "non_existent_cart"
		result.Description = "Get cart that doesn't exist"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Test 2: Add item to non-existent cart
	t.Run("AddItemToNonExistentCart", func(t *testing.T) {
		result := sendFailureTestRequest(
			"POST",
			fmt.Sprintf("%s/shopping-carts/999999999/items", failureTestBaseURL),
			map[string]interface{}{"product_id": 100, "quantity": 1},
			http.StatusNotFound,
		)
		result.TestName = "add_item_to_non_existent_cart"
		result.Description = "Add item to cart that doesn't exist"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Test 3: Invalid cart ID format
	t.Run("InvalidCartIDFormat", func(t *testing.T) {
		result := sendFailureTestRequest(
			"GET",
			fmt.Sprintf("%s/shopping-carts/not_a_number", failureTestBaseURL),
			nil,
			http.StatusBadRequest,
		)
		result.TestName = "invalid_cart_id_format"
		result.Description = "Get cart with non-numeric ID"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Save results
	saveFailureResults(t, allResults, "missing_resources")
}

func TestMalformedData(t *testing.T) {
	fmt.Println("\n╔════════════════════════════════════════╗")
	fmt.Println("║     FAILURE MODE: MALFORMED DATA       ║")
	fmt.Println("╚════════════════════════════════════════╝")
	
	var allResults []FailureTestResult
	
	// Test 1: Extra unexpected fields
	t.Run("ExtraFields", func(t *testing.T) {
		result := sendFailureTestRequest(
			"POST",
			failureTestBaseURL+"/shopping-carts",
			map[string]interface{}{
				"customer_id": 77777,
				"extra_field": "should_be_ignored",
				"another_field": 123,
			},
			http.StatusCreated, // Should succeed and ignore extra fields
		)
		result.TestName = "extra_fields"
		result.Description = "Create cart with extra unexpected fields"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Test 2: Extremely large values
	t.Run("LargeCustomerID", func(t *testing.T) {
		result := sendFailureTestRequest(
			"POST",
			failureTestBaseURL+"/shopping-carts",
			map[string]interface{}{"customer_id": 9223372036854775807}, // Max int64
			http.StatusCreated, // Should succeed
		)
		result.TestName = "large_customer_id"
		result.Description = "Create cart with maximum int64 value"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Test 3: Empty JSON object
	t.Run("EmptyJSON", func(t *testing.T) {
		result := sendFailureTestRequest(
			"POST",
			failureTestBaseURL+"/shopping-carts",
			map[string]interface{}{},
			http.StatusBadRequest,
		)
		result.TestName = "empty_json"
		result.Description = "Send empty JSON object"
		
		fmt.Printf("\n✓ Test: %s\n", result.TestName)
		fmt.Printf("  Expected Status: %d, Got: %d\n", result.ExpectedStatus, result.ActualStatus)
		fmt.Printf("  Response Time: %.2f ms\n", result.ResponseTime)
		fmt.Printf("  Handled correctly: %v\n", result.Success)
		
		allResults = append(allResults, result)
	})
	
	// Save results
	saveFailureResults(t, allResults, "malformed_data")
}

func saveFailureResults(t *testing.T, results []FailureTestResult, testType string) {
	outputFile := fmt.Sprintf("failure_mode_%s_results.json", testType)
	file, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		t.Fatalf("Failed to write JSON: %v", err)
	}
	
	// Calculate success rate
	correctHandling := 0
	for _, r := range results {
		if r.Success {
			correctHandling++
		}
	}
	
	fmt.Printf("\n\n═══════════════════════════════════════════════════════\n")
	fmt.Printf("Summary for %s:\n", testType)
	fmt.Printf("  Total Tests: %d\n", len(results))
	fmt.Printf("  Correctly Handled: %d (%.2f%%)\n", 
		correctHandling, 
		float64(correctHandling)/float64(len(results))*100)
	fmt.Printf("  Results saved to: %s\n", outputFile)
	fmt.Printf("═══════════════════════════════════════════════════════\n")
}
