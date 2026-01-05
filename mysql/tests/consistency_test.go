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

const (
	// Adjust these based on your observations
	maxRetries = 10
	retryDelay = 50 * time.Millisecond
)

var consistencyBaseURL string

// getBaseURLFromTerraform retrieves the ALB URL from Terraform output
func getBaseURLFromTerraform() (string, error) {
	// Get the terraform directory path (one level up from tests directory)
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

// init function to set consistencyBaseURL from Terraform output
func init() {
	url, err := getBaseURLFromTerraform()
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
	consistencyBaseURL = url
}

// Test structures matching your API
type GetCartResponse struct {
	ShoppingCartID int        `json:"shopping_cart_id"`
	CustomerID     int        `json:"customer_id"`
	Items          []CartItem `json:"items"`
}

type CartItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// ConsistencyMetrics tracks consistency observations
type ConsistencyMetrics struct {
	TotalAttempts      int
	ConsistentReads    int
	InconsistentReads  int
	TimeToConsistency  []time.Duration
	FailedReads        int
	mu                 sync.Mutex
}

func (m *ConsistencyMetrics) RecordConsistent(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConsistentReads++
	m.TimeToConsistency = append(m.TimeToConsistency, duration)
}

func (m *ConsistencyMetrics) RecordInconsistent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InconsistentReads++
}

func (m *ConsistencyMetrics) PrintReport() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	fmt.Printf("\n=== Consistency Metrics Report ===\n")
	fmt.Printf("Total Attempts: %d\n", m.TotalAttempts)
	fmt.Printf("Consistent Reads: %d (%.2f%%)\n", 
		m.ConsistentReads, 
		float64(m.ConsistentReads)/float64(m.TotalAttempts)*100)
	fmt.Printf("Inconsistent Reads: %d (%.2f%%)\n", 
		m.InconsistentReads,
		float64(m.InconsistentReads)/float64(m.TotalAttempts)*100)
	fmt.Printf("Failed Reads: %d\n", m.FailedReads)
	
	if len(m.TimeToConsistency) > 0 {
		var total time.Duration
		min := m.TimeToConsistency[0]
		max := m.TimeToConsistency[0]
		
		for _, d := range m.TimeToConsistency {
			total += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}
		
		avg := total / time.Duration(len(m.TimeToConsistency))
		fmt.Printf("\nTime to Consistency:\n")
		fmt.Printf("  Min: %v\n", min)
		fmt.Printf("  Max: %v\n", max)
		fmt.Printf("  Avg: %v\n", avg)
	}
	fmt.Printf("==================================\n\n")
}

// Test 1: Create-then-Read Consistency
func TestCreateThenReadConsistency(t *testing.T) {
	metrics := &ConsistencyMetrics{}
	iterations := 20
	
	fmt.Println("Running Create-then-Read Consistency Test...")
	
	for i := 0; i < iterations; i++ {
		metrics.TotalAttempts++
		customerID := 10000 + i
		
		// Create a cart
		cartID, err := createCart(customerID)
		if err != nil {
			t.Logf("Failed to create cart: %v", err)
			continue
		}
		
		// Immediately try to read it
		start := time.Now()
		cart, err := getCartWithRetry(cartID, maxRetries)
		elapsed := time.Since(start)
		
		if err != nil {
			metrics.FailedReads++
			t.Logf("Failed to read cart %d: %v", cartID, err)
		} else if cart.ShoppingCartID == cartID && cart.CustomerID == customerID {
			metrics.RecordConsistent(elapsed)
			t.Logf("✓ Cart %d consistent (took %v)", cartID, elapsed)
		} else {
			metrics.RecordInconsistent()
			t.Logf("✗ Cart %d data mismatch", cartID)
		}
		
		// Small delay between iterations
		time.Sleep(100 * time.Millisecond)
	}
	
	metrics.PrintReport()
	
	// Save results
	saveConsistencyResults(t, "create_then_read", metrics)
}

// Test 2: Add-Item-then-Read Consistency
func TestAddItemThenReadConsistency(t *testing.T) {
	metrics := &ConsistencyMetrics{}
	iterations := 20
	
	fmt.Println("Running Add-Item-then-Read Consistency Test...")
	
	for i := 0; i < iterations; i++ {
		metrics.TotalAttempts++
		customerID := 20000 + i
		
		// First create a cart
		cartID, err := createCart(customerID)
		if err != nil {
			t.Logf("Failed to create cart: %v", err)
			continue
		}
		
		// Wait a bit for cart creation to propagate
		time.Sleep(200 * time.Millisecond)
		
		// Add an item
		productID := 200 + (i % 10)
		err = addItem(cartID, productID, 5)
		if err != nil {
			t.Logf("Failed to add item: %v", err)
			continue
		}
		
		// Immediately try to read and verify the item
		start := time.Now()
		found := false
		
		for attempt := 0; attempt < maxRetries; attempt++ {
			cart, err := getCart(cartID)
			if err != nil {
				time.Sleep(retryDelay)
				continue
			}
			
			// Check if the item is present
			for _, item := range cart.Items {
				if item.ProductID == productID && item.Quantity == 5 {
					found = true
					elapsed := time.Since(start)
					metrics.RecordConsistent(elapsed)
					t.Logf("✓ Item found in cart %d (took %v)", cartID, elapsed)
					break
				}
			}
			
			if found {
				break
			}
			time.Sleep(retryDelay)
		}
		
		if !found {
			metrics.RecordInconsistent()
			t.Logf("✗ Item not found in cart %d after retries", cartID)
		}
	}
	
	metrics.PrintReport()
	
	// Save results
	saveConsistencyResults(t, "add_item_then_read", metrics)
}

// Test 3: Rapid Concurrent Updates
func TestRapidConcurrentUpdates(t *testing.T) {
	fmt.Println("Running Rapid Concurrent Updates Test...")
	
	// Create a cart first
	cartID, err := createCart(33333)
	if err != nil {
		t.Fatalf("Failed to create cart: %v", err)
	}
	
	// Wait for cart to be available
	time.Sleep(500 * time.Millisecond)
	
	concurrentWriters := 5
	itemsPerWriter := 3
	var wg sync.WaitGroup
	results := make(chan string, concurrentWriters*itemsPerWriter)
	
	// Launch concurrent writers
	for writer := 0; writer < concurrentWriters; writer++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			
			for item := 0; item < itemsPerWriter; item++ {
				productID := (writerID * 10) + item + 300
				quantity := writerID + 1
				
				err := addItem(cartID, productID, quantity)
				if err != nil {
					results <- fmt.Sprintf("Writer %d: ✗ Failed to add item %d", writerID, productID)
				} else {
					results <- fmt.Sprintf("Writer %d: ✓ Added item %d (qty: %d)", writerID, productID, quantity)
				}
				
				time.Sleep(10 * time.Millisecond)
			}
		}(writer)
	}
	
	// Wait for all writers to complete
	wg.Wait()
	close(results)
	
	// Print results
	for result := range results {
		t.Log(result)
	}
	
	// Now read the cart multiple times to observe consistency
	fmt.Println("\nReading cart state multiple times...")
	previousCount := -1
	consistentReads := 0
	
	for i := 0; i < 10; i++ {
		time.Sleep(200 * time.Millisecond)
		cart, err := getCart(cartID)
		if err != nil {
			t.Logf("Read %d: Failed - %v", i+1, err)
			continue
		}
		
		itemCount := len(cart.Items)
		if previousCount != -1 && itemCount == previousCount {
			consistentReads++
		}
		previousCount = itemCount
		t.Logf("Read %d: Found %d items", i+1, itemCount)
		
		// Print item details
		for _, item := range cart.Items {
			t.Logf("  - Product %d: Quantity %d", item.ProductID, item.Quantity)
		}
	}
	
	t.Logf("\nConsistent consecutive reads: %d/9", consistentReads)
}

// Test 4: Write-Read-Write Pattern
func TestWriteReadWritePattern(t *testing.T) {
	fmt.Println("Running Write-Read-Write Pattern Test...")
	
	cartID, err := createCart(44444)
	if err != nil {
		t.Fatalf("Failed to create cart: %v", err)
	}
	
	// Pattern: Write -> Read -> Write based on read -> Verify
	
	// Initial write
	err = addItem(cartID, 500, 10)
	if err != nil {
		t.Fatalf("Failed to add initial item: %v", err)
	}
	
	// Read current state
	time.Sleep(200 * time.Millisecond) // Give some time for propagation
	cart, err := getCart(cartID)
	if err != nil {
		t.Fatalf("Failed to read cart: %v", err)
	}
	
	currentQuantity := 0
	for _, item := range cart.Items {
		if item.ProductID == 500 {
			currentQuantity = item.Quantity
			break
		}
	}
	
	t.Logf("Current quantity for product 500: %d", currentQuantity)
	
	// Update based on read (simulating increment)
	newQuantity := currentQuantity + 5
	err = addItem(cartID, 500, newQuantity)
	if err != nil {
		t.Fatalf("Failed to update item: %v", err)
	}
	
	// Verify the update
	time.Sleep(200 * time.Millisecond)
	cart, err = getCart(cartID)
	if err != nil {
		t.Fatalf("Failed to verify cart: %v", err)
	}
	
	finalQuantity := 0
	for _, item := range cart.Items {
		if item.ProductID == 500 {
			finalQuantity = item.Quantity
			break
		}
	}
	
	t.Logf("Final quantity for product 500: %d (expected: %d)", finalQuantity, newQuantity)
	
	if finalQuantity != newQuantity {
		t.Errorf("Quantity mismatch: expected %d, got %d", newQuantity, finalQuantity)
	}
}

// Helper functions
func createCart(customerID int) (int, error) {
	req := map[string]int{"customer_id": customerID}
	body, _ := json.Marshal(req)
	
	resp, err := http.Post(
		consistencyBaseURL+"/shopping-carts",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("create cart failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	
	cartID := int(result["shopping_cart_id"].(float64))
	return cartID, nil
}

func addItem(cartID int, productID, quantity int) error {
	req := map[string]int{
		"product_id": productID,
		"quantity":   quantity,
	}
	body, _ := json.Marshal(req)
	
	url := fmt.Sprintf("%s/shopping-carts/%d/items", consistencyBaseURL, cartID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add item failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	return nil
}

func getCart(cartID int) (*GetCartResponse, error) {
	url := fmt.Sprintf("%s/shopping-carts/%d", consistencyBaseURL, cartID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cart not found") // Cart not found (eventual consistency)
	}
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get cart failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	var cart GetCartResponse
	if err := json.NewDecoder(resp.Body).Decode(&cart); err != nil {
		return nil, err
	}
	
	return &cart, nil
}

func getCartWithRetry(cartID int, maxRetries int) (*GetCartResponse, error) {
	for i := 0; i < maxRetries; i++ {
		cart, err := getCart(cartID)
		if err == nil {
			return cart, nil
		}
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("cart not available after %d retries", maxRetries)
}

func saveConsistencyResults(t *testing.T, testName string, metrics *ConsistencyMetrics) {
	environment := "aws"
	if strings.Contains(consistencyBaseURL, "localhost") || strings.Contains(consistencyBaseURL, "127.0.0.1") {
		environment = "localstack"
	}
	
	outputFile := fmt.Sprintf("%s_consistency_%s_results.json", environment, testName)
	
	result := map[string]interface{}{
		"test_name":           testName,
		"environment":         environment,
		"total_attempts":      metrics.TotalAttempts,
		"consistent_reads":    metrics.ConsistentReads,
		"inconsistent_reads":  metrics.InconsistentReads,
		"failed_reads":        metrics.FailedReads,
		"consistency_rate":    float64(metrics.ConsistentReads) / float64(metrics.TotalAttempts) * 100,
		"timestamp":           time.Now().Format(time.RFC3339),
	}
	
	file, err := os.Create(outputFile)
	if err != nil {
		t.Logf("Failed to create output file: %v", err)
		return
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(result)
	
	t.Logf("✓ Results saved to: %s", outputFile)
}
