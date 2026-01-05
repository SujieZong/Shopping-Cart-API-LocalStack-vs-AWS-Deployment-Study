#!/bin/bash
# setup-localstack-db.sh - Database initialization script run by LocalStack
# This runs automatically when LocalStack starts

set -e

echo "Initializing database schema..."

# Wait for MySQL to be accessible
sleep 5

# Get RDS endpoint
RDS_ENDPOINT=$(awslocal rds describe-db-instances \
    --db-instance-identifier shopping-cart-mysql \
    --region us-west-2 \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text 2>/dev/null || echo "localhost")

RDS_PORT=$(awslocal rds describe-db-instances \
    --db-instance-identifier shopping-cart-mysql \
    --region us-west-2 \
    --query 'DBInstances[0].Endpoint.Port' \
    --output text 2>/dev/null || echo "43306")

if [ "$RDS_ENDPOINT" != "localhost" ]; then
    echo "Found RDS endpoint: ${RDS_ENDPOINT}:${RDS_PORT}"
    
    # Wait for MySQL to accept connections
    for i in {1..10}; do
        if mysql -h ${RDS_ENDPOINT} -P ${RDS_PORT} -u admin -pMySecurePass123! -e "SELECT 1" 2>/dev/null; then
            echo "MySQL is ready"
            break
        fi
        echo "Waiting for MySQL..."
        sleep 3
    done
    
    # Create database and schema
    mysql -h ${RDS_ENDPOINT} -P ${RDS_PORT} -u admin -pMySecurePass123! << 'EOF'
CREATE DATABASE IF NOT EXISTS shopping_cart_db;
USE shopping_cart_db;

DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS shopping_carts;

CREATE TABLE shopping_carts (
    shopping_cart_id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_customer_id (customer_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE cart_items (
    cart_item_id INT AUTO_INCREMENT PRIMARY KEY,
    shopping_cart_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (shopping_cart_id) 
        REFERENCES shopping_carts(shopping_cart_id) 
        ON DELETE CASCADE,
    UNIQUE KEY unique_cart_product (shopping_cart_id, product_id),
    INDEX idx_shopping_cart_id (shopping_cart_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SHOW TABLES;
EOF
    
    echo "Database schema initialized successfully"
fi
