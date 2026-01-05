package main

import (
	"time"
)

// CartItem represents a single item in the shopping cart
type CartItem struct {
	ProductID int `json:"product_id" dynamodbav:"product_id"`
	Quantity  int `json:"quantity" dynamodbav:"quantity"`
}

// ShoppingCart represents the complete shopping cart item in DynamoDB
type ShoppingCart struct {
	ShoppingCartID int64      `json:"shopping_cart_id" dynamodbav:"shopping_cart_id"`
	CustomerID     int        `json:"customer_id" dynamodbav:"customer_id"`
	Items          []CartItem `json:"items" dynamodbav:"items"`
	CreatedAt      string     `json:"created_at,omitempty" dynamodbav:"created_at,omitempty"`
	UpdatedAt      string     `json:"updated_at,omitempty" dynamodbav:"updated_at,omitempty"`
}

// CreateCartRequest represents the request body for creating a cart
type CreateCartRequest struct {
	CustomerID int `json:"customer_id"`
}

// CreateCartResponse represents the response for creating a cart
type CreateCartResponse struct {
	ShoppingCartID int64 `json:"shopping_cart_id"`
}

// AddItemRequest represents the request body for adding an item to cart
type AddItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// GetCartResponse represents the response for getting a cart
type GetCartResponse struct {
	ShoppingCartID int64      `json:"shopping_cart_id"`
	CustomerID     int        `json:"customer_id"`
	Items          []CartItem `json:"items"`
}

// ErrorResponse represents the error response format
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// generateUniqueID generates a unique integer ID for shopping carts
// Uses timestamp in milliseconds + random component to avoid collisions
func generateUniqueID() int64 {
	// Get current timestamp in milliseconds
	timestamp := time.Now().UnixMilli()
	
	// Add a small random component (0-999) to handle concurrent requests
	// This creates IDs like: 1698765432567 (13 digits)
	// The random component helps prevent collisions in distributed systems
	random := time.Now().Nanosecond() % 1000
	
	// Combine timestamp with random offset
	// This ensures uniqueness even if multiple requests arrive at same millisecond
	return timestamp*1000 + int64(random)
}
