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

var baseURL string

// getBaseURLFromTerraform retrieves the ALB URL from Terraform output
func getBaseURLFromTerraform() (string, error) {
	// Get the terraform directory path (one level up from tests directory)
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
	url, err := getBaseURLFromTerraform()
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
	baseURL = url
}


// Test structures matching your API
type GetCartResponse struct {
	ShoppingCartID int64      `json:"shopping_cart_id"`
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
		var max time.Duration
		min := m.TimeToConsistency[0]
		
		for _, d := range m.TimeToConsistency {
			total += d
			if d > max {
				max = d
			}
			if d < min {
				min = d
			}
		}
		
		avg := total / time.Duration(len(m.TimeToConsistency))
		fmt.Printf("\nTime to Consistency:\n")
		fmt.Printf("  Average: %v\n", avg)
		fmt.Printf("  Min: %v\n", min)
		fmt.Printf("  Max: %v\n", max)
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
		
		// Create a cart
		cartID, err := createCart(12345 + i)
		if err != nil {
			t.Errorf("Failed to create cart: %v", err)
			continue
		}
		
		// Immediately try to read it
		start := time.Now()
		cart, err := getCartWithRetry(cartID, maxRetries)
		elapsed := time.Since(start)
		
		if err != nil {
			metrics.FailedReads++
			t.Logf("Failed to read cart %d after %v: %v", cartID, elapsed, err)
		} else if cart != nil {
			metrics.RecordConsistent(elapsed)
			t.Logf("Cart %d read successfully after %v", cartID, elapsed)
		} else {
			metrics.RecordInconsistent()
			t.Logf("Cart %d not found after %v", cartID, elapsed)
		}
		
		// Small delay between iterations
		time.Sleep(100 * time.Millisecond)
	}
	
	metrics.PrintReport()
}

// Test 2: Add-Item-then-Read Consistency
func TestAddItemThenReadConsistency(t *testing.T) {
	metrics := &ConsistencyMetrics{}
	iterations := 20
	
	fmt.Println("Running Add-Item-then-Read Consistency Test...")
	
	for i := 0; i < iterations; i++ {
		metrics.TotalAttempts++
		
		// First create a cart
		cartID, err := createCart(22345 + i)
		if err != nil {
			t.Errorf("Failed to create cart: %v", err)
			continue
		}
		
		// Wait a bit for cart creation to propagate
		time.Sleep(200 * time.Millisecond)
		
		// Add an item
		err = addItem(cartID, 100+i, 5)
		if err != nil {
			t.Errorf("Failed to add item to cart %d: %v", cartID, err)
			continue
		}
		
		// Immediately try to read and verify the item
		start := time.Now()
		consistent := false
		var elapsed time.Duration
		
		for retry := 0; retry < maxRetries; retry++ {
			cart, err := getCart(cartID)
			elapsed = time.Since(start)
			
			if err == nil && cart != nil && len(cart.Items) > 0 {
				// Check if the item we added is there
				for _, item := range cart.Items {
					if item.ProductID == 100+i && item.Quantity == 5 {
						consistent = true
						break
					}
				}
			}
			
			if consistent {
				break
			}
			
			time.Sleep(retryDelay)
		}
		
		if consistent {
			metrics.RecordConsistent(elapsed)
			t.Logf("Item in cart %d read consistently after %v", cartID, elapsed)
		} else {
			metrics.RecordInconsistent()
			t.Logf("Item in cart %d NOT consistent after %v", cartID, elapsed)
		}
	}
	
	metrics.PrintReport()
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
		go func(w int) {
			defer wg.Done()
			
			for item := 0; item < itemsPerWriter; item++ {
				productID := 1000 + w*100 + item
				quantity := w + 1
				
				err := addItem(cartID, productID, quantity)
				if err != nil {
					results <- fmt.Sprintf("Writer %d: Failed to add product %d: %v", 
						w, productID, err)
				} else {
					results <- fmt.Sprintf("Writer %d: Added product %d with quantity %d", 
						w, productID, quantity)
				}
				
				// Small random delay
				time.Sleep(time.Duration(10+w*5) * time.Millisecond)
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
		cart, err := getCart(cartID)
		if err != nil {
			t.Logf("Read %d: Failed to get cart: %v", i+1, err)
			continue
		}
		
		itemCount := len(cart.Items)
		t.Logf("Read %d: Cart has %d items", i+1, itemCount)
		
		if previousCount >= 0 && itemCount == previousCount {
			consistentReads++
		}
		previousCount = itemCount
		
		// Print item details
		for _, item := range cart.Items {
			t.Logf("  - Product %d: Quantity %d", item.ProductID, item.Quantity)
		}
		
		time.Sleep(100 * time.Millisecond)
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
		t.Fatalf("Failed to read updated cart: %v", err)
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
		t.Errorf("Inconsistent state: expected quantity %d, got %d", newQuantity, finalQuantity)
	}
}

// Helper functions
func createCart(customerID int) (int64, error) {
	req := CreateCartRequest{CustomerID: customerID}
	body, _ := json.Marshal(req)
	
	resp, err := http.Post(
		baseURL+"/shopping-carts",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	
	var result CreateCartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	
	return result.ShoppingCartID, nil
}

func addItem(cartID int64, productID, quantity int) error {
	req := AddItemRequest{
		ProductID: productID,
		Quantity:  quantity,
	}
	body, _ := json.Marshal(req)
	
	url := fmt.Sprintf("%s/shopping-carts/%d/items", baseURL, cartID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	
	return nil
}

func getCart(cartID int64) (*GetCartResponse, error) {
	url := fmt.Sprintf("%s/shopping-carts/%d", baseURL, cartID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Cart not found (eventual consistency)
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	
	var cart GetCartResponse
	if err := json.NewDecoder(resp.Body).Decode(&cart); err != nil {
		return nil, err
	}
	
	return &cart, nil
}

func getCartWithRetry(cartID int64, maxRetries int) (*GetCartResponse, error) {
	for i := 0; i < maxRetries; i++ {
		cart, err := getCart(cartID)
		if err != nil {
			return nil, err
		}
		if cart != nil {
			return cart, nil
		}
		time.Sleep(retryDelay)
	}
	return nil, nil
}

// Run all tests
func main() {
	// You can run these as actual Go tests with: go test -v
	// Or run as a standalone program
	t := &testing.T{}
	
	fmt.Println("Starting DynamoDB Eventual Consistency Tests")
	fmt.Println("================================================")
	
	TestCreateThenReadConsistency(t)
	time.Sleep(1 * time.Second)
	
	TestAddItemThenReadConsistency(t)
	time.Sleep(1 * time.Second)
	
	TestRapidConcurrentUpdates(t)
	time.Sleep(1 * time.Second)
	
	TestWriteReadWritePattern(t)
	
	fmt.Println("\n================================================")
	fmt.Println("Tests Complete!")
}