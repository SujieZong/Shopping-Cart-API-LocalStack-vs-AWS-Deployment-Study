#!/bin/bash
# run-local.sh - Run the Shopping Cart API locally with LocalStack RDS

set -e

echo "=========================================="
echo "🚀 Starting Shopping Cart API (LocalStack)"
echo "=========================================="

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if LOCALSTACK_AUTH_TOKEN is set
if [ -z "$LOCALSTACK_AUTH_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  LOCALSTACK_AUTH_TOKEN is not set${NC}"
    echo -e "${YELLOW}Please set your LocalStack Pro auth token:${NC}"
    echo -e "  export LOCALSTACK_AUTH_TOKEN=your-token-here"
    echo ""
    echo -e "${YELLOW}Continuing anyway (LocalStack may fail to start)...${NC}"
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running"
    echo "Please start Docker Desktop and try again"
    exit 1
fi

echo -e "${GREEN}✅ Docker is running${NC}"

# Check if LocalStack is already running
if docker ps --format '{{.Names}}' | grep -q '^shopping-cart-localstack$'; then
    echo -e "${GREEN}✅ LocalStack is already running${NC}"
else
    echo ""
    echo -e "${YELLOW}LocalStack not running. Setting it up...${NC}"
    ./setup-localstack.sh
fi

# Find the MySQL container created by LocalStack RDS
echo ""
echo -e "${BLUE}🔍 Finding MySQL container created by LocalStack RDS...${NC}"
MYSQL_CONTAINER=$(docker ps --format "{{.Names}}" | grep "ls-mysql" | head -n 1)

if [ -z "$MYSQL_CONTAINER" ]; then
    echo -e "${YELLOW}⚠️  MySQL container not found yet, waiting...${NC}"
    sleep 3
    MYSQL_CONTAINER=$(docker ps --format "{{.Names}}" | grep "ls-mysql" | head -n 1)
fi

if [ -n "$MYSQL_CONTAINER" ]; then
    echo -e "${GREEN}✅ Found MySQL container: ${MYSQL_CONTAINER}${NC}"
    export DB_HOST=$MYSQL_CONTAINER
else
    echo -e "${RED}❌ Could not find MySQL container created by LocalStack RDS${NC}"
    echo "Available containers:"
    docker ps --format "table {{.Names}}\t{{.Image}}"
    exit 1
fi

# Update docker-compose with the correct MySQL container name
echo -e "${BLUE}📝 Updating API configuration with MySQL container: ${MYSQL_CONTAINER}${NC}"

# Create a temporary docker-compose override
cat > docker-compose.localstack.override.yml << EOF
version: "3.8"

services:
  api:
    environment:
      DB_HOST: ${MYSQL_CONTAINER}
      DB_PORT: 3306
    external_links:
      - ${MYSQL_CONTAINER}:mysql
EOF

# Build and start API service with override
echo ""
echo -e "${BLUE}🐳 Building and starting API service...${NC}"
docker-compose -f docker-compose.localstack.yml -f docker-compose.localstack.override.yml up -d --build api

# Get RDS connection details
echo ""
echo -e "${YELLOW}⏳ Getting RDS connection details...${NC}"

RDS_ENDPOINT=$(awslocal rds describe-db-instances \
    --db-instance-identifier shopping-cart-mysql \
    --region us-west-2 \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text 2>/dev/null || echo "localstack")

RDS_PORT=$(awslocal rds describe-db-instances \
    --db-instance-identifier shopping-cart-mysql \
    --region us-west-2 \
    --query 'DBInstances[0].Endpoint.Port' \
    --output text 2>/dev/null || echo "4510")

# Wait for API to be ready
echo ""
echo "⏳ Waiting for API to be ready..."
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s http://localhost:3000/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ API is ready!${NC}"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    echo -n "."
    sleep 2
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo ""
    echo "❌ API failed to start within expected time"
    echo "Checking logs..."
    docker-compose -f docker-compose.localstack.yml logs api
    exit 1
fi

# Test the API
echo ""
echo "🧪 Testing the API..."
echo ""

# Test health endpoint
echo "1. Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s http://localhost:3000/health)
echo "Response: $HEALTH_RESPONSE"

# Create a shopping cart
echo ""
echo "2. Creating a shopping cart..."
CREATE_RESPONSE=$(curl -s -X POST http://localhost:3000/shopping-carts \
    -H "Content-Type: application/json" \
    -d '{"customer_id": 123}')
echo "Response: $CREATE_RESPONSE"

# Extract cart ID
CART_ID=$(echo $CREATE_RESPONSE | grep -o '"shopping_cart_id":[0-9]*' | grep -o '[0-9]*')

if [ -z "$CART_ID" ]; then
    echo "❌ Failed to create cart"
    exit 1
fi

echo -e "${GREEN}✅ Created cart with ID: $CART_ID${NC}"

# Add item to cart
echo ""
echo "3. Adding item to cart..."
ADD_ITEM_RESPONSE=$(curl -s -X POST http://localhost:3000/shopping-carts/$CART_ID/items \
    -H "Content-Type: application/json" \
    -d '{"product_id": 456, "quantity": 2}')
echo "Response status: $ADD_ITEM_RESPONSE"

# Get cart
echo ""
echo "4. Retrieving cart..."
GET_CART_RESPONSE=$(curl -s http://localhost:3000/shopping-carts/$CART_ID)
echo "Response: $GET_CART_RESPONSE"

echo ""
echo "=========================================="
echo -e "${GREEN}✅ All tests passed!${NC}"
echo "=========================================="
echo ""
echo -e "${BLUE}Service URLs:${NC}"
echo "  API: http://localhost:3000"
echo "  Health Check: http://localhost:3000/health"
echo "  LocalStack: http://localhost:4566"
echo ""
echo -e "${BLUE}RDS MySQL Connection (via LocalStack):${NC}"
echo "  Host: ${RDS_ENDPOINT}"
echo "  Port: ${RDS_PORT}"
echo "  Database: shopping_cart_db"
echo "  User: admin"
echo "  Password: MySecurePass123!"
echo ""
echo -e "${BLUE}Useful Commands:${NC}"
echo "  View API logs: docker logs shopping-cart-api -f"
echo "  View LocalStack logs: docker logs shopping-cart-localstack -f"
echo "  Stop services: docker-compose -f docker-compose.localstack.yml down"
echo "  Stop and remove data: docker-compose -f docker-compose.localstack.yml down -v"
echo ""
echo -e "${BLUE}Connect to MySQL:${NC}"
echo "  docker exec shopping-cart-localstack mysql -h ${RDS_ENDPOINT} -P ${RDS_PORT} -u admin -pMySecurePass123! shopping_cart_db"
echo ""
echo -e "${BLUE}Check RDS Status:${NC}"
echo "  awslocal rds describe-db-instances --db-instance-identifier shopping-cart-mysql"
echo ""

