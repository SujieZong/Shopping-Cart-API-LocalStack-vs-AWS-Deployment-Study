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

var coldStartBaseURL string

// ColdStartResult stores cold start measurement results
type ColdStartResult struct {
	Environment           string  `json:"environment"` // "localstack" or "aws"
	InfrastructureUpTime  float64 `json:"infrastructure_up_time_seconds"`
	FirstSuccessTime      float64 `json:"first_success_time_seconds"`
	TotalColdStartTime    float64 `json:"total_cold_start_time_seconds"`
	FirstRequestAttempts  int     `json:"first_request_attempts"`
	HealthCheckLatency    float64 `json:"health_check_latency_ms"`
	FirstCreateLatency    float64 `json:"first_create_latency_ms"`
	ConsecutiveSuccesses  int     `json:"consecutive_successes"`
	Timestamp             string  `json:"timestamp"`
}

func init() {
	url, err := getColdStartBaseURL()
	if err != nil {
		// Fallback to environment variable if terraform output fails
		url = os.Getenv("API_URL")
		if url == "" {
			url = "http://localhost:3000"
		}
		fmt.Printf("Using API_URL from environment variable or default: %s\n", url)
	} else {
		fmt.Printf("Successfully retrieved base URL from Terraform: %s\n", url)
	}
	coldStartBaseURL = url
}

func getColdStartBaseURL() (string, error) {
	terraformDir := filepath.Join("..", "terraform")
	cmd := exec.Command("terraform", "output", "-raw", "alb_url")
	cmd.Dir = terraformDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get terraform output: %v", err)
	}
	
	url := strings.TrimSpace(string(output))
	
	// Check if the output contains terraform warnings or is invalid
	if url == "" || strings.Contains(url, "Warning:") || strings.Contains(url, "╷") || strings.Contains(url, "\x1b[") {
		return "", fmt.Errorf("terraform output returned invalid or empty URL")
	}
	
	return url, nil
}

// waitForServiceReady attempts to connect to the service and measures time to first success
func waitForServiceReady(client *http.Client, maxAttempts int, retryDelay time.Duration) (time.Duration, int, error) {
	fmt.Println("\nWaiting for service to become ready...")
	startTime := time.Now()
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("Attempt %d/%d...\n", attempt, maxAttempts)
		
		resp, err := client.Get(coldStartBaseURL + "/health")
		if err == nil && resp != nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				elapsed := time.Since(startTime)
				fmt.Printf("✓ Service ready after %d attempts (%.2f seconds)\n", attempt, elapsed.Seconds())
				return elapsed, attempt, nil
			}
		}
		
		if attempt < maxAttempts {
			time.Sleep(retryDelay)
		}
	}
	
	return time.Since(startTime), maxAttempts, fmt.Errorf("service not ready after %d attempts", maxAttempts)
}

// measureFirstRequest attempts the first actual API request
func measureFirstRequest(client *http.Client) (time.Duration, bool) {
	fmt.Println("\nAttempting first create cart request...")
	
	req := map[string]int{"customer_id": 99999}
	body, _ := json.Marshal(req)
	
	start := time.Now()
	resp, err := client.Post(
		coldStartBaseURL+"/shopping-carts",
		"application/json",
		bytes.NewBuffer(body),
	)
	elapsed := time.Since(start)
	
	if err != nil {
		fmt.Printf("✗ First request failed: %v\n", err)
		return elapsed, false
	}
	defer resp.Body.Close()
	
	success := resp.StatusCode == http.StatusCreated
	if success {
		fmt.Printf("✓ First request succeeded (%.2f ms)\n", float64(elapsed.Microseconds())/1000.0)
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("✗ First request returned status %d: %s\n", resp.StatusCode, string(bodyBytes))
	}
	
	return elapsed, success
}

// verifyStability makes several consecutive requests to verify service is stable
func verifyStability(client *http.Client, numRequests int) int {
	fmt.Printf("\nVerifying stability with %d consecutive requests...\n", numRequests)
	successCount := 0
	
	for i := 1; i <= numRequests; i++ {
		req := map[string]int{"customer_id": 90000 + i}
		body, _ := json.Marshal(req)
		
		resp, err := client.Post(
			coldStartBaseURL+"/shopping-carts",
			"application/json",
			bytes.NewBuffer(body),
		)
		
		if err == nil && resp != nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusCreated {
				successCount++
				fmt.Printf("  Request %d/%d: ✓\n", i, numRequests)
			} else {
				fmt.Printf("  Request %d/%d: ✗ (status %d)\n", i, numRequests, resp.StatusCode)
			}
		} else {
			fmt.Printf("  Request %d/%d: ✗ (error)\n", i, numRequests)
		}
		
		time.Sleep(100 * time.Millisecond)
	}
	
	fmt.Printf("Stability: %d/%d successful\n", successCount, numRequests)
	return successCount
}

func TestColdStart(t *testing.T) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	fmt.Println("\n╔════════════════════════════════════════╗")
	fmt.Println("║      COLD START MEASUREMENT TEST      ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Printf("\nBase URL: %s\n", coldStartBaseURL)
	fmt.Println("\nNOTE: This test should be run immediately after:")
	fmt.Println("  - AWS: terraform apply")
	fmt.Println("  - LocalStack: ./run-local.sh")
	
	// Phase 1: Wait for service to respond
	readyTime, attempts, err := waitForServiceReady(client, 60, 5*time.Second)
	if err != nil {
		t.Fatalf("Service never became ready: %v", err)
	}
	
	// Phase 2: Measure first actual request
	firstReqTime, firstSuccess := measureFirstRequest(client)
	if !firstSuccess {
		t.Logf("First request failed, trying again...")
		// Try a few more times
		for i := 0; i < 3; i++ {
			time.Sleep(2 * time.Second)
			firstReqTime, firstSuccess = measureFirstRequest(client)
			if firstSuccess {
				break
			}
		}
	}
	
	// Phase 3: Verify stability
	consecutiveSuccesses := verifyStability(client, 5)
	
	// Phase 4: Measure health check latency
	healthStart := time.Now()
	resp, err := client.Get(coldStartBaseURL + "/health")
	healthLatency := time.Since(healthStart)
	if resp != nil {
		resp.Body.Close()
	}
	
	// Calculate total cold start time
	totalColdStart := readyTime
	if firstSuccess {
		totalColdStart = readyTime + firstReqTime
	}
	
	// Determine environment
	environment := "aws"
	if strings.Contains(coldStartBaseURL, "localhost") || strings.Contains(coldStartBaseURL, "127.0.0.1") {
		environment = "localstack"
	}
	
	result := ColdStartResult{
		Environment:          environment,
		InfrastructureUpTime: readyTime.Seconds(),
		FirstSuccessTime:     firstReqTime.Seconds(),
		TotalColdStartTime:   totalColdStart.Seconds(),
		FirstRequestAttempts: attempts,
		HealthCheckLatency:   float64(healthLatency.Microseconds()) / 1000.0,
		FirstCreateLatency:   float64(firstReqTime.Microseconds()) / 1000.0,
		ConsecutiveSuccesses: consecutiveSuccesses,
		Timestamp:            time.Now().Format(time.RFC3339),
	}
	
	// Print summary
	fmt.Println("\n\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  COLD START SUMMARY                           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\nEnvironment: %s\n", result.Environment)
	fmt.Printf("Infrastructure Up Time: %.2f seconds\n", result.InfrastructureUpTime)
	fmt.Printf("Time to First Successful Request: %.2f seconds\n", result.FirstSuccessTime)
	fmt.Printf("Total Cold Start Time: %.2f seconds\n", result.TotalColdStartTime)
	fmt.Printf("Attempts Until Ready: %d\n", result.FirstRequestAttempts)
	fmt.Printf("Health Check Latency: %.2f ms\n", result.HealthCheckLatency)
	fmt.Printf("First Create Latency: %.2f ms\n", result.FirstCreateLatency)
	fmt.Printf("Consecutive Successes: %d/5\n", result.ConsecutiveSuccesses)
	
	// Save results
	outputFile := fmt.Sprintf("cold_start_%s_results.json", environment)
	file, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("Failed to encode results: %v", err)
	}
	
	fmt.Printf("\n✓ Results saved to: %s\n", outputFile)
	
	if result.ConsecutiveSuccesses < 3 {
		t.Logf("Warning: Service may not be fully stable (only %d/5 consecutive successes)", result.ConsecutiveSuccesses)
	}
}

// TestManualColdStartInstructions provides instructions for manual cold start testing
func TestManualColdStartInstructions(t *testing.T) {
	t.Skip("This is an instructional test - skip by default")
	
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║            MANUAL COLD START TESTING INSTRUCTIONS             ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	
	fmt.Println("\n📋 AWS Environment:")
	fmt.Println("   1. Ensure infrastructure is destroyed: cd ../terraform && terraform destroy")
	fmt.Println("   2. Deploy fresh infrastructure: terraform apply")
	fmt.Println("   3. Start timer when you run 'terraform apply'")
	fmt.Println("   4. Run this test: go test -v -run TestColdStart")
	fmt.Println("   5. Note the total cold start time")
	
	fmt.Println("\n📋 LocalStack Environment:")
	fmt.Println("   1. Stop services: docker-compose down")
	fmt.Println("   2. Start timer when you run: ./run-local.sh")
	fmt.Println("   3. Export API_URL: export API_URL=http://localhost:3000")
	fmt.Println("   4. Run this test: go test -v -run TestColdStart")
	fmt.Println("   5. Note the total cold start time")
	
	fmt.Println("\n📊 Comparison Metrics:")
	fmt.Println("   - Infrastructure Up Time: How long until health check passes")
	fmt.Println("   - First Request Time: Latency of the very first API call")
	fmt.Println("   - Total Cold Start: From infrastructure up to stable service")
	fmt.Println("   - Consecutive Successes: Stability indicator")
}
