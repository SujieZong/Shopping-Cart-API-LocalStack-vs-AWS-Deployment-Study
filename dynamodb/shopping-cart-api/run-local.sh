#!/bin/bash
set -e

# Local Development Script
# Starts LocalStack infrastructure AND the shopping cart API

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== Starting Local Development Environment ===${NC}"

# Step 1: Start LocalStack if not already running
if ! docker ps --format '{{.Names}}' | grep -q '^localstack$'; then
    echo -e "${YELLOW}LocalStack not running. Starting it now...${NC}"
    ./setup-localstack.sh
else
    echo -e "${GREEN}✓ LocalStack is already running${NC}"
    
    # Verify DynamoDB is available
    if ! curl -s http://localhost:4566/_localstack/health | grep -q '"dynamodb": *"\(available\|running\)"'; then
        echo -e "${RED}ERROR: LocalStack DynamoDB is not available${NC}"
        echo -e "${YELLOW}Try restarting: docker restart localstack${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ DynamoDB is available${NC}"
fi

# Step 2: Set environment variables for the API
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-west-2
export DYNAMODB_TABLE_NAME=ShoppingCarts
export PORT=8080

echo -e "\n${BLUE}Environment Configuration:${NC}"
echo -e "  AWS_ENDPOINT_URL: ${AWS_ENDPOINT_URL}"
echo -e "  AWS_REGION: ${AWS_REGION}"
echo -e "  DYNAMODB_TABLE_NAME: ${DYNAMODB_TABLE_NAME}"
echo -e "  PORT: ${PORT}"

# Step 3: Check if API is already running
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo -e "\n${YELLOW}⚠️  Port 8080 is already in use${NC}"
    echo -e "${YELLOW}Kill the existing process? (yes/no)${NC}"
    read -p "> " kill_existing
    if [ "$kill_existing" = "yes" ]; then
        echo -e "${BLUE}Killing process on port 8080...${NC}"
        lsof -ti:8080 | xargs kill -9
        sleep 2
    else
        echo -e "${RED}Cannot start API - port 8080 is in use${NC}"
        exit 1
    fi
fi

# Step 4: Start the API
echo -e "\n${GREEN}Starting Shopping Cart API on port 8080...${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo -e "${BLUE}Logs will be shown below:${NC}\n"

# Run the application (foreground so you can see logs and stop with Ctrl+C)
go run *.go
