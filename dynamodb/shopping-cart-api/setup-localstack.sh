#!/bin/bash
set -e

# LocalStack Infrastructure Setup Script
# Run this to set up DynamoDB on LocalStack for local testing

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== Setting up LocalStack Infrastructure ===${NC}"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

# Check if LocalStack container is already running
if docker ps --format '{{.Names}}' | grep -q '^localstack$'; then
    echo -e "${YELLOW}LocalStack container is already running${NC}"
    read -p "Do you want to recreate it? (yes/no): " recreate
    if [ "$recreate" = "yes" ]; then
        echo -e "${BLUE}Stopping and removing existing LocalStack container...${NC}"
        docker stop localstack
        docker rm localstack
    else
        echo -e "${YELLOW}Using existing LocalStack container${NC}"
    fi
fi

# Start LocalStack if not running
if ! docker ps --format '{{.Names}}' | grep -q '^localstack$'; then
    echo -e "\n${BLUE}Starting LocalStack...${NC}"
    docker run -d \
      --name localstack \
      -p 4566:4566 \
      -e SERVICES=dynamodb \
      -e DEBUG=1 \
      -e DOCKER_HOST=unix:///var/run/docker.sock \
      -v /var/run/docker.sock:/var/run/docker.sock \
      localstack/localstack

    # Wait for LocalStack to be ready
    echo -e "\n${BLUE}Waiting for LocalStack to be ready...${NC}"
    sleep 10
    
    # Check if LocalStack is healthy
    max_attempts=30
    attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:4566/_localstack/health | grep -q '"dynamodb": "available"'; then
            echo -e "${GREEN}✓ LocalStack is ready${NC}"
            break
        fi
        attempt=$((attempt + 1))
        echo -e "${YELLOW}Waiting for LocalStack... ($attempt/$max_attempts)${NC}"
        sleep 2
    done
    
    if [ $attempt -eq $max_attempts ]; then
        echo -e "${RED}ERROR: LocalStack failed to start${NC}"
        docker logs localstack
        exit 1
    fi
fi

# Set environment variables for AWS CLI
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-west-2
ENDPOINT_URL=http://localhost:4566

# Check if table already exists
echo -e "\n${BLUE}Checking if ShoppingCarts table exists...${NC}"
if aws dynamodb describe-table \
    --endpoint-url $ENDPOINT_URL \
    --table-name ShoppingCarts \
    --region $AWS_DEFAULT_REGION > /dev/null 2>&1; then
    echo -e "${YELLOW}ShoppingCarts table already exists${NC}"
    read -p "Do you want to recreate it? (yes/no): " recreate_table
    if [ "$recreate_table" = "yes" ]; then
        echo -e "${BLUE}Deleting existing table...${NC}"
        aws dynamodb delete-table \
            --endpoint-url $ENDPOINT_URL \
            --table-name ShoppingCarts \
            --region $AWS_DEFAULT_REGION > /dev/null
        echo -e "${GREEN}✓ Table deleted${NC}"
        sleep 2
    else
        echo -e "${YELLOW}Using existing table${NC}"
        aws dynamodb list-tables --endpoint-url $ENDPOINT_URL --region $AWS_DEFAULT_REGION
        echo -e "\n${GREEN}=== LocalStack Infrastructure Ready! ===${NC}"
        echo -e "\n${YELLOW}To run the application locally:${NC}"
        echo -e "  export AWS_ENDPOINT_URL=http://localhost:4566"
        echo -e "  export AWS_ACCESS_KEY_ID=test"
        echo -e "  export AWS_SECRET_ACCESS_KEY=test"
        echo -e "  export AWS_REGION=us-west-2"
        echo -e "  export DYNAMODB_TABLE_NAME=ShoppingCarts"
        echo -e "  go run *.go"
        exit 0
    fi
fi

# Create DynamoDB table
echo -e "\n${GREEN}Creating DynamoDB table...${NC}"
aws dynamodb create-table \
  --endpoint-url $ENDPOINT_URL \
  --table-name ShoppingCarts \
  --attribute-definitions \
    AttributeName=shopping_cart_id,AttributeType=N \
  --key-schema \
    AttributeName=shopping_cart_id,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region $AWS_DEFAULT_REGION > /dev/null

echo -e "${GREEN}✓ DynamoDB table created${NC}"

# Verify table creation
echo -e "\n${BLUE}Verifying table creation...${NC}"
aws dynamodb describe-table \
  --endpoint-url $ENDPOINT_URL \
  --table-name ShoppingCarts \
  --region $AWS_DEFAULT_REGION \
  --query 'Table.{Name:TableName,Status:TableStatus,HashKey:KeySchema[0].AttributeName,RangeKey:KeySchema[1].AttributeName}' \
  --output table 2>/dev/null || echo -e "${GREEN}Table created successfully${NC}"

# List all tables
echo -e "\n${BLUE}All DynamoDB tables:${NC}"
aws dynamodb list-tables --endpoint-url $ENDPOINT_URL --region $AWS_DEFAULT_REGION

echo -e "\n${GREEN}=== LocalStack Infrastructure Ready! ===${NC}"
echo -e "\n${YELLOW}Environment variables for your application:${NC}"
echo -e "  ${BLUE}AWS_ENDPOINT_URL${NC}=http://localhost:4566"
echo -e "  ${BLUE}AWS_ACCESS_KEY_ID${NC}=test"
echo -e "  ${BLUE}AWS_SECRET_ACCESS_KEY${NC}=test"
echo -e "  ${BLUE}AWS_REGION${NC}=us-west-2"
echo -e "  ${BLUE}DYNAMODB_TABLE_NAME${NC}=ShoppingCarts"

echo -e "\n${YELLOW}To run the application locally:${NC}"
echo -e "  ${GREEN}export AWS_ENDPOINT_URL=http://localhost:4566${NC}"
echo -e "  ${GREEN}export AWS_ACCESS_KEY_ID=test${NC}"
echo -e "  ${GREEN}export AWS_SECRET_ACCESS_KEY=test${NC}"
echo -e "  ${GREEN}export AWS_REGION=us-west-2${NC}"
echo -e "  ${GREEN}export DYNAMODB_TABLE_NAME=ShoppingCarts${NC}"
echo -e "  ${GREEN}go run *.go${NC}"

echo -e "\n${YELLOW}Or use the run-local.sh script:${NC}"
echo -e "  ${GREEN}./run-local.sh${NC}"
