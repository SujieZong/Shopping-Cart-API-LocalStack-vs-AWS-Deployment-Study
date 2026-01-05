package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ShoppingCartHandler handles HTTP requests for shopping cart operations
type ShoppingCartHandler struct {
	repo *ShoppingCartRepository
}

// NewShoppingCartHandler creates a new handler instance
func NewShoppingCartHandler(repo *ShoppingCartRepository) *ShoppingCartHandler {
	return &ShoppingCartHandler{
		repo: repo,
	}
}

// CreateCart handles POST /shopping-carts
// Creates a new shopping cart for a customer
func (h *ShoppingCartHandler) CreateCart(c *gin.Context) {
	var req CreateCartRequest

	// Parse and validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("WARN: Invalid request body: %v", err)
		HandleError(c, ValidationError("Invalid request body"))
		return
	}

	// Validate customer_id
	if req.CustomerID <= 0 {
		log.Printf("WARN: Invalid customer_id: %d", req.CustomerID)
		HandleError(c, ValidationError("customer_id must be a positive integer"))
		return
	}

	// Create the cart
	cart, err := h.repo.CreateCart(c.Request.Context(), req.CustomerID)
	if err != nil {
		HandleError(c, err)
		return
	}

	// Return response with 201 Created
	response := CreateCartResponse{
		ShoppingCartID: cart.ShoppingCartID,
	}

	log.Printf("INFO: Created cart %d for customer %d", cart.ShoppingCartID, req.CustomerID)
	c.JSON(http.StatusCreated, response)
}

// AddItem handles POST /shopping-carts/{shoppingCartId}/items
// Adds or updates an item in the shopping cart
func (h *ShoppingCartHandler) AddItem(c *gin.Context) {
	// Parse shopping cart ID from URL parameter
	cartIDStr := c.Param("shoppingCartId")
	cartID, err := strconv.ParseInt(cartIDStr, 10, 64)
	if err != nil || cartID <= 0 {
		log.Printf("WARN: Invalid shopping cart ID: %s", cartIDStr)
		HandleError(c, ValidationError("Invalid shopping cart ID"))
		return
	}

	// Parse and validate request body
	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("WARN: Invalid request body: %v", err)
		HandleError(c, ValidationError("Invalid request body"))
		return
	}

	// Validate product_id
	if req.ProductID <= 0 {
		log.Printf("WARN: Invalid product_id: %d", req.ProductID)
		HandleError(c, ValidationError("product_id must be a positive integer"))
		return
	}

	// Validate quantity
	if req.Quantity <= 0 {
		log.Printf("WARN: Invalid quantity: %d", req.Quantity)
		HandleError(c, ValidationError("quantity must be a positive integer"))
		return
	}

	// Additional validation: reasonable quantity limit
	if req.Quantity > 10000 {
		log.Printf("WARN: Quantity too large: %d", req.Quantity)
		HandleError(c, ValidationError("quantity cannot exceed 10,000"))
		return
	}

	// Add or update the item
	err = h.repo.AddOrUpdateItem(c.Request.Context(), cartID, req.ProductID, req.Quantity)
	if err != nil {
		HandleError(c, err)
		return
	}

	// Return 204 No Content on success
	log.Printf("INFO: Updated cart %d with product %d (quantity: %d)", cartID, req.ProductID, req.Quantity)
	c.Status(http.StatusNoContent)
}

// GetCart handles GET /shopping-carts/{shoppingCartId}
// Retrieves a shopping cart by ID
func (h *ShoppingCartHandler) GetCart(c *gin.Context) {
	// Parse shopping cart ID from URL parameter
	cartIDStr := c.Param("shoppingCartId")
	cartID, err := strconv.ParseInt(cartIDStr, 10, 64)
	if err != nil || cartID <= 0 {
		log.Printf("WARN: Invalid shopping cart ID: %s", cartIDStr)
		HandleError(c, ValidationError("Invalid shopping cart ID"))
		return
	}

	// Get the cart from repository
	cart, err := h.repo.GetCart(c.Request.Context(), cartID)
	if err != nil {
		HandleError(c, err)
		return
	}

	// Ensure items is empty array, not null
	if cart.Items == nil {
		cart.Items = []CartItem{}
	}

	// Create response in the exact required format
	response := GetCartResponse{
		ShoppingCartID: cart.ShoppingCartID,
		CustomerID:     cart.CustomerID,
		Items:          cart.Items,
	}

	log.Printf("INFO: Retrieved cart %d with %d items", cartID, len(cart.Items))
	c.JSON(http.StatusOK, response)
}

// HealthCheck handles GET /health
// Returns service health status
func (h *ShoppingCartHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "shopping-cart-api",
	})
}
