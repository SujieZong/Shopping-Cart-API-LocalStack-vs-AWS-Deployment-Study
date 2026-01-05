package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration from environment
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		tableName = "ShoppingCarts" // Default table name
	}

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-west-2" // Default region
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	// Initialize AWS SDK
	ctx := context.Background()
	
	// Check if LocalStack endpoint is configured
	localstackEndpoint := os.Getenv("AWS_ENDPOINT_URL")
	
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(awsRegion),
	)
	if err != nil {
		log.Fatalf("FATAL: Failed to load AWS config: %v", err)
	}

	// Create DynamoDB client with optional endpoint override
	var dynamoClient *dynamodb.Client
	if localstackEndpoint != "" {
		log.Printf("INFO: Using LocalStack endpoint: %s", localstackEndpoint)
		dynamoClient = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = &localstackEndpoint
		})
	} else {
		log.Printf("INFO: Using AWS DynamoDB in region: %s", awsRegion)
		dynamoClient = dynamodb.NewFromConfig(cfg)
	}

	// Initialize repository and handler
	repo := NewShoppingCartRepository(dynamoClient, tableName)
	handler := NewShoppingCartHandler(repo)

	// Set up Gin router
	router := gin.Default()

	// Configure Gin to trust proxy headers (for ALB)
	router.SetTrustedProxies(nil)

	// CORS middleware (if needed for frontend)
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", handler.HealthCheck)

	// Shopping cart endpoints
	router.POST("/shopping-carts", handler.CreateCart)
	router.POST("/shopping-carts/:shoppingCartId/items", handler.AddItem)
	router.GET("/shopping-carts/:shoppingCartId", handler.GetCart)

	// Start the server
	log.Printf("INFO: Starting shopping cart API server on port %s", port)
	log.Printf("INFO: Using DynamoDB table: %s", tableName)
	log.Printf("INFO: AWS Region: %s", awsRegion)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("FATAL: Failed to start server: %v", err)
	}
}
