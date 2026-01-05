package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	ErrCartNotFound = errors.New("shopping cart not found")
	ErrInvalidInput = errors.New("invalid input")
)

// ShoppingCartRepository handles all DynamoDB operations for shopping carts
type ShoppingCartRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewShoppingCartRepository creates a new repository instance
func NewShoppingCartRepository(client *dynamodb.Client, tableName string) *ShoppingCartRepository {
	return &ShoppingCartRepository{
		client:    client,
		tableName: tableName,
	}
}

// CreateCart creates a new shopping cart in DynamoDB
func (r *ShoppingCartRepository) CreateCart(ctx context.Context, customerID int) (*ShoppingCart, error) {
	// Generate unique shopping cart ID
	shoppingCartID := generateUniqueID()
	
	// Create the cart object
	cart := &ShoppingCart{
		ShoppingCartID: shoppingCartID,
		CustomerID:     customerID,
		Items:          []CartItem{}, // Empty items array initially
		CreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}

	// Marshal the cart into DynamoDB attribute values
	item, err := attributevalue.MarshalMap(cart)
	if err != nil {
		log.Printf("ERROR: Failed to marshal cart: %v", err)
		return nil, fmt.Errorf("failed to marshal cart: %w", err)
	}

	// Put the item into DynamoDB
	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		log.Printf("ERROR: Failed to create cart in DynamoDB: %v", err)
		return nil, fmt.Errorf("failed to create cart: %w", err)
	}

	log.Printf("INFO: Created cart with ID: %d for customer: %d", shoppingCartID, customerID)
	return cart, nil
}

// GetCart retrieves a shopping cart by ID
func (r *ShoppingCartRepository) GetCart(ctx context.Context, shoppingCartID int64) (*ShoppingCart, error) {
	// Create the key for GetItem
	key, err := attributevalue.MarshalMap(map[string]interface{}{
		"shopping_cart_id": shoppingCartID,
	})
	if err != nil {
		log.Printf("ERROR: Failed to marshal key: %v", err)
		return nil, fmt.Errorf("failed to marshal key: %w", err)
	}

	// Get the item from DynamoDB
	input := &dynamodb.GetItemInput{
		TableName:      aws.String(r.tableName),
		Key:            key,
		//ConsistentRead: aws.Bool(true), // Enable strong consistency for read
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		log.Printf("ERROR: Failed to get cart from DynamoDB: %v", err)
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// Check if item exists
	if result.Item == nil {
		log.Printf("WARN: Cart not found with ID: %d", shoppingCartID)
		return nil, ErrCartNotFound
	}

	// Unmarshal the result into ShoppingCart
	var cart ShoppingCart
	err = attributevalue.UnmarshalMap(result.Item, &cart)
	if err != nil {
		log.Printf("ERROR: Failed to unmarshal cart: %v", err)
		return nil, fmt.Errorf("failed to unmarshal cart: %w", err)
	}

	// Ensure items is not nil (should be empty array instead)
	if cart.Items == nil {
		cart.Items = []CartItem{}
	}

	log.Printf("INFO: Retrieved cart ID: %d with %d items", shoppingCartID, len(cart.Items))
	return &cart, nil
}

// AddOrUpdateItem adds a new item to the cart or updates quantity if it already exists
// If product_id exists, OVERRIDE the quantity (not add)
// Uses DynamoDB Transactions for atomic operations to prevent lost updates
func (r *ShoppingCartRepository) AddOrUpdateItem(ctx context.Context, shoppingCartID int64, productID, quantity int) error {
    // Implement retry logic for transaction conflicts
    maxRetries := 10
    baseDelay := 20 * time.Millisecond
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        if attempt > 0 {
            // Exponential backoff with jitter
            backoffMultiplier := 1 << uint(attempt-1)
            delay := time.Duration(int64(baseDelay) * int64(backoffMultiplier))
            jitter := time.Duration(float64(delay) * 0.3 * (0.5 + 0.5*float64(time.Now().UnixNano()%100)/100.0))
            time.Sleep(delay + jitter)
            log.Printf("INFO: Retry attempt %d for cart %d, product %d after transaction conflict", attempt+1, shoppingCartID, productID)
        }
        
        // First, get the current cart to check if it exists and get current version
        cart, err := r.GetCart(ctx, shoppingCartID)
        if err != nil {
            return err
        }

        // Initialize items if nil
        if cart.Items == nil {
            cart.Items = []CartItem{}
        }

        // Store the original version for optimistic locking check
        originalVersion := cart.UpdatedAt

        // Find if the product already exists in items array
        existingIndex := -1
        for i, item := range cart.Items {
            if item.ProductID == productID {
                existingIndex = i
                break
            }
        }

        // Update the items array
        if existingIndex >= 0 {
            cart.Items[existingIndex].Quantity = quantity
        } else {
            cart.Items = append(cart.Items, CartItem{
                ProductID: productID,
                Quantity:  quantity,
            })
        }

        // Update the timestamp with nanosecond precision
        newTimestamp := time.Now().UTC().Format(time.RFC3339Nano)
        cart.UpdatedAt = newTimestamp

        // Create the key using attributevalue.MarshalMap for consistency
        key, err := attributevalue.MarshalMap(map[string]interface{}{
            "shopping_cart_id": shoppingCartID,
        })
        if err != nil {
            log.Printf("ERROR: Failed to marshal key: %v", err)
            return fmt.Errorf("failed to marshal key: %w", err)
        }

        // Marshal items (handle empty list properly)
        itemsAttr, err := attributevalue.Marshal(cart.Items)
        if err != nil {
            log.Printf("ERROR: Failed to marshal items: %v", err)
            return fmt.Errorf("failed to marshal items: %w", err)
        }

        // Marshal the updated timestamp
        timestampAttr, err := attributevalue.Marshal(newTimestamp)
        if err != nil {
            log.Printf("ERROR: Failed to marshal timestamp: %v", err)
            return fmt.Errorf("failed to marshal timestamp: %w", err)
        }

        // Marshal the original version for optimistic locking
        oldVersionAttr, err := attributevalue.Marshal(originalVersion)
        if err != nil {
            log.Printf("ERROR: Failed to marshal old version: %v", err)
            return fmt.Errorf("failed to marshal old version: %w", err)
        }

        // Use DynamoDB Transaction for atomic update with optimistic locking
        // This ensures ACID guarantees and prevents lost updates
        transactInput := &dynamodb.TransactWriteItemsInput{
            TransactItems: []types.TransactWriteItem{
                {
                    Update: &types.Update{
                        TableName: aws.String(r.tableName),
                        Key:       key,
                        UpdateExpression: aws.String("SET #items = :items, updated_at = :timestamp"),
                        ExpressionAttributeNames: map[string]string{
                            "#items": "items",
                        },
                        ExpressionAttributeValues: map[string]types.AttributeValue{
                            ":items":      itemsAttr,
                            ":timestamp":  timestampAttr,
                            ":oldVersion": oldVersionAttr,
                        },
                        ConditionExpression: aws.String("attribute_exists(shopping_cart_id) AND updated_at = :oldVersion"),
                    },
                },
            },
        }

        _, err = r.client.TransactWriteItems(ctx, transactInput)
        if err != nil {
            // Handle transaction cancellation (optimistic lock failure)
            var txCanceled *types.TransactionCanceledException
            if errors.As(err, &txCanceled) {
                if attempt < maxRetries-1 {
                    // Transaction conflict - another update occurred, retry
                    log.Printf("INFO: Transaction conflict for cart %d, product %d - retrying...", shoppingCartID, productID)
                    continue
                }
                // Max retries reached
                log.Printf("WARN: Max retries (%d) reached for cart %d, product %d after transaction conflicts", maxRetries, shoppingCartID, productID)
                return fmt.Errorf("concurrent update conflict: max retries reached after %d attempts", maxRetries)
            }
            log.Printf("ERROR: Failed to execute transaction for cart %d: %v", shoppingCartID, err)
            return fmt.Errorf("failed to update cart: %w", err)
        }

        log.Printf("INFO: Successfully updated cart %d with product %d, quantity %d (attempt %d, final count: %d items)", 
            shoppingCartID, productID, quantity, attempt+1, len(cart.Items))
        return nil
    }
    
    return fmt.Errorf("failed to update cart after %d retries due to persistent transaction conflicts", maxRetries)
}
// Alternative implementation using DynamoDB's list_append (more complex but avoids GetItem)
// This is commented out but shows a different approach
/*
func (r *ShoppingCartRepository) AddOrUpdateItemOptimized(ctx context.Context, shoppingCartID int64, productID, quantity int) error {
	// This approach would require custom logic to:
	// 1. Remove old entry if exists (using REMOVE on specific index)
	// 2. Append new entry
	// However, it's complex because you need to know the index beforehand
	// The GetItem approach above is simpler and still performant
	
	// For production, if GetItem latency becomes an issue, you could:
	// - Use TransactWriteItems for atomic read-modify-write
	// - Implement optimistic locking with version numbers
	// - Use DynamoDB Streams to handle consistency
}
*/
