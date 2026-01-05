#!/bin/bash
set -e

# Script to create DynamoDB table in LocalStack after docker-compose up

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== Creating DynamoDB Table in LocalStack ===${NC}"

# Wait for LocalStack to be ready
echo -e "${BLUE}Waiting for LocalStack...${NC}"
sleep 5

# Set environment variables
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-west-2
ENDPOINT_URL=http://localhost:4566

# Create table
echo -e "\n${GREEN}Creating ShoppingCarts table...${NC}"
aws dynamodb create-table \
  --endpoint-url $ENDPOINT_URL \
  --table-name ShoppingCarts \
  --attribute-definitions \
    AttributeName=userId,AttributeType=S \
    AttributeName=itemId,AttributeType=S \
  --key-schema \
    AttributeName=userId,KeyType=HASH \
    AttributeName=itemId,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --region $AWS_DEFAULT_REGION > /dev/null

echo -e "${GREEN}✓ Table created successfully${NC}"

# Verify
aws dynamodb list-tables --endpoint-url $ENDPOINT_URL --region $AWS_DEFAULT_REGION
