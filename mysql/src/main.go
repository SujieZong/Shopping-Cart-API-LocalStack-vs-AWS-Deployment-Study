package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// ============================================================================
// Data Models
// ============================================================================

type ShoppingCart struct {
	ShoppingCartID int        `json:"shopping_cart_id"`
	CustomerID     int        `json:"customer_id"`
	Items          []CartItem `json:"items"`
}

type CartItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type CreateCartRequest struct {
	CustomerID int `json:"customer_id" binding:"required,min=1"`
}

type AddItemRequest struct {
	ProductID int `json:"product_id" binding:"required,min=1"`
	Quantity  int `json:"quantity" binding:"required,min=1"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ============================================================================
// Database Connection
// ============================================================================

var db *sql.DB

func initDB() {
	var err error

	// Get database config from environment variables
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "admin"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "MySecurePass123!"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "shopping_cart_db"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}

	// Build DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// Open database connection with connection pool settings
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)                 // Maximum number of open connections
	db.SetMaxIdleConns(5)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum lifetime of a connection

	// Test connection
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("[OK] Connected to MySQL database")
	log.Printf("Connection pool: MaxOpen=%d, MaxIdle=%d", 10, 5)
}

// Safe initSchema - won't fail if tables exist
func initSchema() {
	log.Println("Checking and initializing database schema...")

	// Create shopping_carts table - CREATE TABLE IF NOT EXISTS is safe
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS shopping_carts (
			shopping_cart_id INT AUTO_INCREMENT PRIMARY KEY,
			customer_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_customer_id (customer_id),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)

	if err != nil {
		log.Printf("Warning: Failed to create shopping_carts table: %v", err)
		// Don't exit, continue running
	} else {
		log.Println("[OK] Table shopping_carts is ready")
	}

	// Create cart_items table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cart_items (
			cart_item_id INT AUTO_INCREMENT PRIMARY KEY,
			shopping_cart_id INT NOT NULL,
			product_id INT NOT NULL,
			quantity INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (shopping_cart_id) 
				REFERENCES shopping_carts(shopping_cart_id) 
				ON DELETE CASCADE,
			UNIQUE KEY unique_cart_product (shopping_cart_id, product_id),
			INDEX idx_shopping_cart_id (shopping_cart_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)

	if err != nil {
		log.Printf("Warning: Failed to create cart_items table: %v", err)
		// Don't exit, continue running
	} else {
		log.Println("[OK] Table cart_items is ready")
	}

	log.Println("[OK] Database schema initialization complete")
	// No error return, this function always succeeds
}

// ============================================================================
// Handlers
// ============================================================================

// POST /shopping-carts
func createCart(c *gin.Context) {
	startTime := time.Now()

	var req CreateCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "customer_id is required and must be a number",
		})
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to create shopping cart",
		})
		return
	}
	defer tx.Rollback()

	// Insert into shopping_carts table
	result, err := tx.Exec(
		"INSERT INTO shopping_carts (customer_id) VALUES (?)",
		req.CustomerID,
	)
	if err != nil {
		log.Printf("Failed to insert cart: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to create shopping cart",
		})
		return
	}

	cartID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to get last insert ID: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to create shopping cart",
		})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to create shopping cart",
		})
		return
	}

	responseTime := time.Since(startTime).Milliseconds()
	log.Printf("POST /shopping-carts - %dms - Cart ID: %d", responseTime, cartID)

	// Return shopping_cart_id as INTEGER (important!)
	c.JSON(http.StatusCreated, gin.H{
		"shopping_cart_id": int(cartID),
	})
}

// POST /shopping-carts/:id/items
func addItem(c *gin.Context) {
	startTime := time.Now()

	cartID := c.Param("id")

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "product_id and quantity (> 0) are required",
		})
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to add items to cart",
		})
		return
	}
	defer tx.Rollback()

	// Check if cart exists
	var exists int
	err = tx.QueryRow(
		"SELECT COUNT(*) FROM shopping_carts WHERE shopping_cart_id = ?",
		cartID,
	).Scan(&exists)

	if err != nil || exists == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: "Shopping cart not found",
		})
		return
	}

	// Insert or update item (overwrite logic, not accumulate)
	_, err = tx.Exec(`
		INSERT INTO cart_items (shopping_cart_id, product_id, quantity) 
		VALUES (?, ?, ?) 
		ON DUPLICATE KEY UPDATE quantity = VALUES(quantity)`,
		cartID, req.ProductID, req.Quantity,
	)
	if err != nil {
		log.Printf("Failed to upsert item: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to add items to cart",
		})
		return
	}

	// Update cart's updated_at timestamp
	_, err = tx.Exec(
		"UPDATE shopping_carts SET updated_at = CURRENT_TIMESTAMP WHERE shopping_cart_id = ?",
		cartID,
	)
	if err != nil {
		log.Printf("Failed to update timestamp: %v", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to add items to cart",
		})
		return
	}

	responseTime := time.Since(startTime).Milliseconds()
	log.Printf("POST /shopping-carts/%s/items - %dms", cartID, responseTime)

	c.Status(http.StatusNoContent)
}

// GET /shopping-carts/:id
func getCart(c *gin.Context) {
	startTime := time.Now()

	cartID := c.Param("id")

	// Query cart with items using JOIN
	rows, err := db.Query(`
		SELECT 
			c.shopping_cart_id,
			c.customer_id,
			COALESCE(ci.product_id, 0) as product_id,
			COALESCE(ci.quantity, 0) as quantity
		FROM shopping_carts c
		LEFT JOIN cart_items ci ON c.shopping_cart_id = ci.shopping_cart_id
		WHERE c.shopping_cart_id = ?`,
		cartID,
	)
	if err != nil {
		log.Printf("Failed to query cart: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "Failed to retrieve shopping cart",
		})
		return
	}
	defer rows.Close()

	var cart *ShoppingCart
	items := make([]CartItem, 0)

	for rows.Next() {
		var (
			shoppingCartID int
			customerID     int
			productID      int
			quantity       int
		)

		err = rows.Scan(&shoppingCartID, &customerID, &productID, &quantity)
		if err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		// First row, initialize cart
		if cart == nil {
			cart = &ShoppingCart{
				ShoppingCartID: shoppingCartID,
				CustomerID:     customerID,
				Items:          []CartItem{},
			}
		}

		// Add item if exists (productID > 0 means there's an item)
		if productID > 0 {
			items = append(items, CartItem{
				ProductID: productID,
				Quantity:  quantity,
			})
		}
	}

	// Cart not found
	if cart == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: "Shopping cart not found",
		})
		return
	}

	cart.Items = items

	responseTime := time.Since(startTime).Milliseconds()
	log.Printf("GET /shopping-carts/%s - %dms", cartID, responseTime)

	c.JSON(http.StatusOK, cart)
}

// Health check endpoint
func healthCheck(c *gin.Context) {
	// Check database connection
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"service": "shopping-cart-mysql",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "shopping-cart-mysql",
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	// Initialize database
	initDB()
	defer db.Close()

	// Initialize schema - don't check error, function handles internally
	initSchema()

	// Setup Gin router
	router := gin.Default()

	// Add middleware for logging
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	// Routes
	router.GET("/health", healthCheck)
	router.POST("/shopping-carts", createCart)
	router.POST("/shopping-carts/:id/items", addItem)
	router.GET("/shopping-carts/:id", getCart)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("[START] Shopping Cart MySQL API starting on port %s", port)
	log.Fatal(router.Run(":" + port))
}
