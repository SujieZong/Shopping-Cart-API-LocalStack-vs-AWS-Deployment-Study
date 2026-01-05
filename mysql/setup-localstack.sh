#!/bin/bash
# setup-localstack.sh - Initialize LocalStack with RDS MySQL instance

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=========================================="
echo "🚀 Setting Up LocalStack with RDS MySQL"
echo -e "==========================================${NC}"

# Check if LOCALSTACK_AUTH_TOKEN is set
if [ -z "$LOCALSTACK_AUTH_TOKEN" ]; then
    echo -e "${RED}ERROR: LOCALSTACK_AUTH_TOKEN is not set${NC}"
    echo -e "${YELLOW}Please set your LocalStack Pro auth token:${NC}"
    echo -e "  export LOCALSTACK_AUTH_TOKEN=your-token-here"
    echo -e "${YELLOW}Or add it to your shell profile (~/.zshrc or ~/.bash_profile)${NC}"
    exit 1
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Error: Docker is not running${NC}"
    echo "Please start Docker Desktop and try again"
    exit 1
fi

# Check if awslocal is installed
if ! command -v awslocal &> /dev/null; then
    echo -e "${RED}❌ Error: awslocal is not installed${NC}"
    echo -e "${YELLOW}Installing awscli-local...${NC}"
    pip3 install awscli-local || {
        echo -e "${RED}Failed to install awscli-local${NC}"
        echo -e "${YELLOW}Please install manually: pip3 install awscli-local${NC}"
        exit 1
    }
    echo -e "${GREEN}✅ awscli-local installed successfully${NC}"
fi

echo -e "${GREEN}✅ Docker is running${NC}"
echo -e "${GREEN}✅ LocalStack Pro token detected${NC}"
echo -e "${GREEN}✅ awslocal CLI is available${NC}"

# Check if LocalStack is already running on port 4566
echo ""
echo -e "${BLUE}🔍 Checking for existing LocalStack container...${NC}"

EXISTING_LOCALSTACK=$(docker ps --filter "publish=4566" --format "{{.Names}}" | head -n 1)

if [ -n "$EXISTING_LOCALSTACK" ]; then
    echo -e "${YELLOW}⚠️  Found existing LocalStack container: ${EXISTING_LOCALSTACK}${NC}"
    
    # Check if it has RDS enabled
    echo -e "${BLUE}🔍 Checking if RDS is available in existing LocalStack...${NC}"
    RDS_STATUS=$(curl -s http://localhost:4566/_localstack/health | grep -o '"rds":"[^"]*"' | cut -d'"' -f4)
    
    if [ "$RDS_STATUS" = "available" ] || [ "$RDS_STATUS" = "running" ]; then
        echo -e "${GREEN}✅ Existing LocalStack has RDS enabled, reusing it...${NC}"
        LOCALSTACK_CONTAINER="$EXISTING_LOCALSTACK"
    else
        echo -e "${YELLOW}⚠️  Existing LocalStack doesn't have RDS enabled${NC}"
        echo -e "${YELLOW}🛑 Stopping existing LocalStack container to start one with RDS...${NC}"
        
        docker stop "$EXISTING_LOCALSTACK" 2>/dev/null || true
        docker rm "$EXISTING_LOCALSTACK" 2>/dev/null || true
        
        echo -e "${BLUE}🐳 Starting new LocalStack with RDS support...${NC}"
        docker-compose -f docker-compose.localstack.yml up -d localstack
        
        LOCALSTACK_CONTAINER="shopping-cart-localstack"
        
        # Wait for LocalStack to be ready
        echo ""
        echo -e "${YELLOW}⏳ Waiting for LocalStack to be ready...${NC}"
        MAX_ATTEMPTS=60
        ATTEMPT=0
        
        while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
            if curl -s http://localhost:4566/_localstack/health | grep -q '"rds"'; then
                echo -e "${GREEN}✅ LocalStack is ready!${NC}"
                break
            fi
            ATTEMPT=$((ATTEMPT + 1))
            echo -n "."
            sleep 2
        done
        
        if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
            echo ""
            echo -e "${RED}❌ LocalStack failed to start within expected time${NC}"
            echo "Checking logs..."
            docker logs $LOCALSTACK_CONTAINER
            exit 1
        fi
    fi
else
    echo -e "${YELLOW}No existing LocalStack found. Starting new instance...${NC}"
    
    # Clean up any stopped containers
    echo -e "${YELLOW}🧹 Cleaning up stopped containers...${NC}"
    docker-compose -f docker-compose.localstack.yml down 2>/dev/null || true
    
    # Start LocalStack
    echo ""
    echo -e "${BLUE}🐳 Starting LocalStack...${NC}"
    docker-compose -f docker-compose.localstack.yml up -d localstack
    
    LOCALSTACK_CONTAINER="shopping-cart-localstack"
    
    # Wait for LocalStack to be ready
    echo ""
    echo -e "${YELLOW}⏳ Waiting for LocalStack to be ready...${NC}"
    MAX_ATTEMPTS=60
    ATTEMPT=0
    
    while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
        if curl -s http://localhost:4566/_localstack/health | grep -q '"rds"'; then
            echo -e "${GREEN}✅ LocalStack is ready!${NC}"
            break
        fi
        ATTEMPT=$((ATTEMPT + 1))
        echo -n "."
        sleep 2
    done
    
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo ""
        echo -e "${RED}❌ LocalStack failed to start within expected time${NC}"
        echo "Checking logs..."
        docker logs $LOCALSTACK_CONTAINER
        exit 1
    fi
fi

# Check if RDS instance already exists
echo ""
echo -e "${BLUE}🔍 Checking for existing RDS instance...${NC}"

if awslocal rds describe-db-instances --db-instance-identifier shopping-cart-mysql --region us-west-2 2>/dev/null | grep -q "shopping-cart-mysql"; then
    echo -e "${GREEN}✅ RDS instance 'shopping-cart-mysql' already exists${NC}"
    echo -e "${YELLOW}♻️  Reusing existing RDS instance...${NC}"
else
    echo -e "${BLUE}📊 Creating new RDS MySQL instance...${NC}"
    
    awslocal rds create-db-instance \
        --db-instance-identifier shopping-cart-mysql \
        --db-instance-class db.t3.micro \
        --engine mysql \
        --master-username admin \
        --master-user-password MySecurePass123! \
        --allocated-storage 20 \
        --db-name shopping_cart_db \
        --region us-west-2 || {
        echo -e "${YELLOW}⚠️  RDS instance creation returned an error, checking if it exists...${NC}"
    }
    
    # Wait for RDS instance to be available
    echo ""
    echo -e "${YELLOW}⏳ Waiting for RDS instance to be available (this takes ~15 seconds)...${NC}"
    ATTEMPT=0
    MAX_ATTEMPTS=30
    
    while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
        STATUS=$(awslocal rds describe-db-instances \
            --db-instance-identifier shopping-cart-mysql \
            --region us-west-2 \
            --query 'DBInstances[0].DBInstanceStatus' \
            --output text 2>/dev/null || echo "unknown")
        
        if [ "$STATUS" = "available" ]; then
            echo -e "${GREEN}✅ RDS instance is available!${NC}"
            break
        fi
        ATTEMPT=$((ATTEMPT + 1))
        echo -n "."
        sleep 3
    done
    
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo ""
        echo -e "${YELLOW}⚠️  RDS instance taking longer than expected, but continuing...${NC}"
    fi
fi

# Get RDS endpoint
echo ""
echo -e "${BLUE}🔍 Getting RDS endpoint information...${NC}"

RDS_ENDPOINT=$(awslocal rds describe-db-instances \
    --db-instance-identifier shopping-cart-mysql \
    --region us-west-2 \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text 2>/dev/null || echo "")

RDS_PORT=$(awslocal rds describe-db-instances \
    --db-instance-identifier shopping-cart-mysql \
    --region us-west-2 \
    --query 'DBInstances[0].Endpoint.Port' \
    --output text 2>/dev/null || echo "43306")

if [ -z "$RDS_ENDPOINT" ] || [ "$RDS_ENDPOINT" = "None" ]; then
    echo -e "${YELLOW}⚠️  RDS endpoint not immediately available, waiting...${NC}"
    sleep 10
    RDS_ENDPOINT=$(awslocal rds describe-db-instances \
        --db-instance-identifier shopping-cart-mysql \
        --region us-west-2 \
        --query 'DBInstances[0].Endpoint.Address' \
        --output text)
fi

echo -e "${GREEN}✅ RDS Endpoint: ${RDS_ENDPOINT}:${RDS_PORT}${NC}"

# Find the MySQL container created by LocalStack RDS
echo ""
echo -e "${BLUE}🔍 Finding MySQL container created by LocalStack RDS...${NC}"
MYSQL_CONTAINER=$(docker ps --filter "ancestor=mysql:8.0" --format "{{.Names}}" | grep "ls-mysql" | head -n 1)

if [ -z "$MYSQL_CONTAINER" ]; then
    echo -e "${YELLOW}⚠️  MySQL container not found yet, waiting...${NC}"
    sleep 5
    MYSQL_CONTAINER=$(docker ps --filter "ancestor=mysql:8.0" --format "{{.Names}}" | grep "ls-mysql" | head -n 1)
fi

if [ -n "$MYSQL_CONTAINER" ]; then
    echo -e "${GREEN}✅ Found MySQL container: ${MYSQL_CONTAINER}${NC}"
else
    echo -e "${RED}❌ Could not find MySQL container${NC}"
    echo -e "${YELLOW}Checking all MySQL containers:${NC}"
    docker ps | grep mysql
    MYSQL_CONTAINER="mysql"
fi

# Export for other scripts
export RDS_ENDPOINT
export RDS_PORT
export LOCALSTACK_CONTAINER
export MYSQL_CONTAINER

# Initialize database schema
echo ""
echo -e "${BLUE}📝 Initializing database schema...${NC}"

# Wait for MySQL to be fully ready
echo -e "${YELLOW}⏳ Waiting for MySQL to accept connections...${NC}"
sleep 5

# Try to initialize schema directly on the MySQL container
docker exec $MYSQL_CONTAINER mysql \
    -u admin \
    -pMySecurePass123! \
    shopping_cart_db \
    -e "$(cat src/db/setup.sql)" 2>/dev/null && {
    echo -e "${GREEN}✅ Database schema initialized successfully${NC}"
} || {
    echo -e "${YELLOW}⚠️  Schema initialization had issues. Will be created when API starts.${NC}"
}

echo ""
echo -e "${GREEN}=========================================="
echo "✅ LocalStack RDS Setup Complete!"
echo -e "==========================================${NC}"
echo ""
echo -e "${BLUE}LocalStack Information:${NC}"
echo "  LocalStack URL: http://localhost:4566"
echo "  Health Check: http://localhost:4566/_localstack/health"
echo ""
echo -e "${BLUE}RDS MySQL Information:${NC}"
echo "  RDS Endpoint: ${RDS_ENDPOINT}:${RDS_PORT}"
echo "  MySQL Container: ${MYSQL_CONTAINER}"
echo "  Database: shopping_cart_db"
echo "  Username: admin"
echo "  Password: MySecurePass123!"
echo ""
echo -e "${BLUE}Connect to MySQL:${NC}"
echo "  docker exec -it ${MYSQL_CONTAINER} mysql -u admin -pMySecurePass123! shopping_cart_db"
echo ""
echo -e "${BLUE}Useful Commands:${NC}"
echo "  Check RDS status: awslocal rds describe-db-instances --db-instance-identifier shopping-cart-mysql"
echo "  View LocalStack logs: docker logs $LOCALSTACK_CONTAINER -f"
echo "  View MySQL logs: docker logs $MYSQL_CONTAINER -f"
echo "  List RDS instances: awslocal rds describe-db-instances --region us-west-2"
echo ""
echo -e "${YELLOW}Next step: Run ./run-local.sh to start the API${NC}"
echo -e "${YELLOW}Note: The API will connect to MySQL container: ${MYSQL_CONTAINER}${NC}"
