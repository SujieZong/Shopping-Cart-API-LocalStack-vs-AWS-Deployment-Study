# LocalStack Quick Start Guide

Run the Shopping Cart API locally using LocalStack (no AWS account needed!)

## 🚀 Quick Start (3 Commands)

```bash
# 1. Set up LocalStack and create DynamoDB table
./setup-localstack.sh

# 2. Run the application
./run-local.sh

# 3. Test the API
curl http://localhost:8080/health
```

## 📋 What Gets Created

- **LocalStack Container**: Runs AWS services locally on port 4566
- **DynamoDB Table**: `ShoppingCarts` table with userId and itemId keys
- **API Server**: Runs on port 8080

## 🧪 Test the API

### Health Check

```bash
curl http://localhost:8080/health
```

### Create a Shopping Cart

```bash
curl -X POST http://localhost:8080/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 12345}'
```

### Add Item to Cart

```bash
curl -X POST http://localhost:8080/shopping-carts/user-12345/items \
  -H "Content-Type: application/json" \
  -d '{
    "item_id": "item-001",
    "product_name": "Laptop",
    "quantity": 1,
    "price": 999.99
  }'
```

### Get Shopping Cart

```bash
curl http://localhost:8080/shopping-carts/user-12345
```

## 🔍 Inspect DynamoDB Data

```bash
# List all tables
aws dynamodb list-tables \
  --endpoint-url http://localhost:4566 \
  --region us-west-2

# Scan table contents
aws dynamodb scan \
  --table-name ShoppingCarts \
  --endpoint-url http://localhost:4566 \
  --region us-west-2
```

## 🛑 Stop and Clean Up

```bash
# Stop LocalStack
docker stop localstack
docker rm localstack

# The application will stop automatically (Ctrl+C if running)
```

## 🐳 Alternative: Docker Compose

If you prefer running everything in containers:

```bash
# Start services
docker-compose -f docker-compose.localstack.yml up -d

# Wait 10 seconds, then create table
sleep 10
./setup-localstack-table.sh

# View logs
docker-compose -f docker-compose.localstack.yml logs -f

# Stop services
docker-compose -f docker-compose.localstack.yml down
```

## 🔧 Troubleshooting

**LocalStack won't start:**

```bash
docker ps -a  # Check if already running
docker logs localstack  # Check logs
```

**Can't connect to DynamoDB:**

```bash
curl http://localhost:4566/_localstack/health
```

**Port 4566 already in use:**

```bash
lsof -i :4566  # Find what's using the port
docker stop localstack  # Stop existing container
```

## 📚 Full Documentation

See [DEPLOYMENT_GUIDE.md](../DEPLOYMENT_GUIDE.md) for complete LocalStack and AWS deployment instructions.
