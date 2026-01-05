# Shopping Cart API - DynamoDB Implementation (Go)

A high-performance REST API for managing shopping carts using AWS DynamoDB, implemented in Go.

## 🚀 Quick Start

### LocalStack (Local Development - Recommended for Testing)

```bash
./setup-localstack.sh  # Set up LocalStack + DynamoDB
./run-local.sh         # Run the API locally
```

📖 See [LOCALSTACK_QUICKSTART.md](LOCALSTACK_QUICKSTART.md) for details

### AWS (Production Deployment)

```bash
./setup-infrastructure.sh  # Create AWS resources with Terraform
./deploy.sh               # Deploy to ECS
```

📖 See [../DEPLOYMENT_GUIDE.md](../DEPLOYMENT_GUIDE.md) for complete guide

## 🏗️ Architecture

- **Language**: Go 1.21
- **Framework**: Gin Web Framework
- **Database**: AWS DynamoDB
- **SDK**: AWS SDK for Go v2
- **Deployment**: Docker + ECS Fargate (AWS) or LocalStack (Local)

## 📋 Features

### Unique ID Generation

- Timestamp-based (milliseconds) + random component
- Collision-resistant in distributed environments
- Generates integer IDs (not UUIDs)
- Format: 13-digit number (e.g., `1698765432567`)

### API Endpoints

#### 1. Create Shopping Cart

```bash
POST /shopping-carts
Content-Type: application/json

{
  "customer_id": 12345
}

# Response: 201 Created
{
  "shopping_cart_id": 1698765432567
}
```

#### 2. Add/Update Item

```bash
POST /shopping-carts/{shoppingCartId}/items
Content-Type: application/json

{
  "product_id": 100,
  "quantity": 2
}

# Response: 204 No Content
```

**Behavior:**

- If `product_id` exists → **OVERRIDE** quantity (not add)
- If `product_id` doesn't exist → append to items array
- Returns 404 if cart doesn't exist

#### 3. Get Shopping Cart

```bash
GET /shopping-carts/{shoppingCartId}

# Response: 200 OK
{
  "shopping_cart_id": 1698765432567,
  "customer_id": 12345,
  "items": [
    {"product_id": 100, "quantity": 2},
    {"product_id": 101, "quantity": 3}
  ]
}

# Empty cart returns:
{
  "shopping_cart_id": 1698765432567,
  "customer_id": 12345,
  "items": []
}

# Cart not found: 404 Not Found
{
  "error": "CART_NOT_FOUND",
  "message": "Shopping cart not found"
}
```

#### 4. Health Check

```bash
GET /health

# Response: 200 OK
{
  "status": "healthy",
  "service": "shopping-cart-api"
}
```

## 🔧 Error Handling

All errors return this format:

```json
{
  "error": "ERROR_CODE",
  "message": "Human readable description"
}
```

### Error Codes

| HTTP Status | Error Code           | Description                     |
| ----------- | -------------------- | ------------------------------- |
| 400         | `BAD_REQUEST`        | Invalid request format          |
| 400         | `INVALID_INPUT`      | Invalid field values            |
| 400         | `VALIDATION_ERROR`   | DynamoDB validation failed      |
| 404         | `CART_NOT_FOUND`     | Shopping cart doesn't exist     |
| 404         | `RESOURCE_NOT_FOUND` | DynamoDB table not found        |
| 429         | `THROTTLING_ERROR`   | Rate limit exceeded             |
| 500         | `INTERNAL_ERROR`     | Unexpected server error         |
| 500         | `DATABASE_ERROR`     | DynamoDB operation failed       |
| 503         | `THROTTLING_ERROR`   | Service temporarily unavailable |

### DynamoDB Exception Mapping

```go
// Handled exceptions:
- ResourceNotFoundException       → 404 (table/resource not found)
- ProvisionedThroughputExceeded  → 503 (throttling)
- ValidationException             → 400 (invalid data)
- ConditionalCheckFailed          → 404 (cart doesn't exist)
- InternalServerError             → 500 (database error)
- RequestLimitExceeded            → 429 (too many requests)
- TransactionConflictException    → 409 (conflict)
```

## 🚀 Local Development

### Prerequisites

- Go 1.21 or later
- AWS credentials configured
- DynamoDB table `ShoppingCarts` created

### Setup

1. **Install dependencies**

```bash
cd shopping-cart-api
go mod download
```

2. **Set environment variables**

```bash
export AWS_REGION=us-west-2
export DYNAMODB_TABLE_NAME=ShoppingCarts
export PORT=8080
```

3. **Run locally**

```bash
go run .
```

4. **Test the API**

```bash
# Health check
curl http://localhost:8080/health

# Create cart
curl -X POST http://localhost:8080/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 12345}'

# Add item (replace {id} with actual cart ID)
curl -X POST http://localhost:8080/shopping-carts/{id}/items \
  -H "Content-Type: application/json" \
  -d '{"product_id": 100, "quantity": 2}'

# Get cart
curl http://localhost:8080/shopping-carts/{id}
```

## 🐳 Docker Build

### Build image

```bash
docker build -t shopping-cart-api .
```

### Run container

```bash
docker run -p 8080:8080 \
  -e AWS_REGION=us-west-2 \
  -e DYNAMODB_TABLE_NAME=ShoppingCarts \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  shopping-cart-api
```

## 📦 Deployment (ECS Fargate)

### Push to ECR

```bash
# Authenticate Docker to ECR
aws ecr get-login-password --region us-west-2 | \
  docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-west-2.amazonaws.com

# Tag image
docker tag shopping-cart-api:latest <account-id>.dkr.ecr.us-west-2.amazonaws.com/shopping-cart-api:latest

# Push image
docker push <account-id>.dkr.ecr.us-west-2.amazonaws.com/shopping-cart-api:latest
```

### Deploy with Terraform

```bash
cd ../terraform
terraform init
terraform plan
terraform apply
```

The Terraform configuration will:

- Create DynamoDB table `ShoppingCarts`
- Set up ECS task definition with environment variables
- Configure IAM roles for DynamoDB access
- Deploy to Fargate with ALB

## 🧪 Testing

### Unit Tests (TODO)

```bash
go test ./... -v
```

### Load Testing

See `../tests/` directory for load testing scripts that generate:

- 50 carts with customer IDs 1000-1049
- Products 100-109 (cycling)
- Quantities 1-5 (cycling)

## 📊 Performance

### Expected Latencies (with DynamoDB)

| Operation       | DynamoDB API         | Expected Time |
| --------------- | -------------------- | ------------- |
| Create Cart     | PutItem              | 5-10ms        |
| Add/Update Item | GetItem + UpdateItem | 10-20ms       |
| Get Cart        | GetItem              | 5-10ms        |

**Total API Response Time:** < 50ms (meets requirements)

### ID Generation Performance

- **Method**: `time.Now().UnixMilli() * 1000 + (nanosecond % 1000)`
- **Collision Probability**: Negligible (handles 1000 requests per millisecond)
- **Thread-Safe**: Yes

## 🔐 Security

### IAM Permissions Required

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["dynamodb:PutItem", "dynamodb:GetItem", "dynamodb:UpdateItem"],
      "Resource": "arn:aws:dynamodb:us-west-2:*:table/ShoppingCarts"
    }
  ]
}
```

### Environment Variables

- `DYNAMODB_TABLE_NAME`: DynamoDB table name
- `AWS_REGION`: AWS region
- `PORT`: HTTP server port (default: 8080)

## 📁 Project Structure

```
shopping-cart-api/
├── main.go           # Entry point, server setup
├── models.go         # Data structures, ID generation
├── repository.go     # DynamoDB operations
├── handlers.go       # HTTP request handlers
├── errors.go         # Error handling, DynamoDB exception mapping
├── go.mod            # Go dependencies
├── go.sum            # Dependency checksums
├── Dockerfile        # Container image
└── README.md         # This file
```

## 🐛 Troubleshooting

### Cart not found (404)

- Verify cart ID is correct (integer)
- Check DynamoDB table has the item
- Ensure IAM permissions are correct

### Throttling errors (503)

- DynamoDB table using on-demand billing should auto-scale
- Check CloudWatch metrics for `UserErrors`
- Consider implementing exponential backoff

### Connection errors

- Verify AWS credentials are set
- Check security groups allow ECS tasks to reach DynamoDB
- Ensure VPC endpoints configured (if using private subnets)

### Items array is null instead of empty

- This should not happen - code explicitly sets `items: []`
- Check JSON marshaling in handlers.go

## 📝 Notes

- All IDs are **integers** (not strings)
- Empty items array is `[]` (not `null`)
- Quantity updates **override** (not add/subtract)
- Timestamps use ISO 8601 format (RFC3339)
- Logs use structured format with severity levels

## 🔗 Related Documentation

- [DynamoDB Schema Design](../SCHEMA_DESIGN.md)
- [API Specification](../README.md)
- [Terraform Infrastructure](../terraform/main.tf)
