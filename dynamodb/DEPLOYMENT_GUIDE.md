# Shopping Cart API - Deployment Guide

This guide covers two deployment approaches for the Shopping Cart API with DynamoDB:

1. **LocalStack** - For local development and testing
2. **AWS** - For production deployment on AWS infrastructure

> **Note:** All AWS resources are prefixed with `dynamodb-` to avoid conflicts with other deployments (e.g., MySQL). See [TERRAFORM_CHANGES.md](TERRAFORM_CHANGES.md) for details.

---

## Table of Contents

- [LocalStack Deployment (Local Development)](#localstack-deployment-local-development)
- [AWS Deployment (Production)](#aws-deployment-production)
- [Testing the API](#testing-the-api)
- [Troubleshooting](#troubleshooting)

---

## LocalStack Deployment (Local Development)

LocalStack provides a fully functional local AWS cloud stack for development and testing without incurring AWS costs.

### Prerequisites

- Docker Desktop installed and running
- Go 1.21+ installed
- AWS CLI installed
- Bash shell

### Option 1: Quick Start (Recommended)

```bash
cd dynamodb/shopping-cart-api

# 1. Set up LocalStack and create DynamoDB table
chmod +x setup-localstack.sh
./setup-localstack.sh

# 2. Run the application
chmod +x run-local.sh
./run-local.sh
```

The API will be available at: `http://localhost:8080`

### Option 2: Docker Compose (Full Container Setup)

```bash
cd dynamodb/shopping-cart-api

# 1. Start LocalStack and the application
docker-compose -f docker-compose.localstack.yml up -d

# 2. Wait for services to be ready (about 10 seconds)
sleep 10

# 3. Create the DynamoDB table
chmod +x setup-localstack-table.sh
./setup-localstack-table.sh

# 4. Check logs
docker-compose -f docker-compose.localstack.yml logs -f shopping-cart-api
```

The API will be available at: `http://localhost:8080`

### Option 3: Manual Setup

```bash
cd dynamodb/shopping-cart-api

# 1. Start LocalStack
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=dynamodb \
  -e DEBUG=1 \
  localstack/localstack

# 2. Wait for LocalStack to be ready
sleep 10

# 3. Create DynamoDB table
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-west-2

aws dynamodb create-table \
  --endpoint-url http://localhost:4566 \
  --table-name ShoppingCarts \
  --attribute-definitions \
    AttributeName=userId,AttributeType=S \
    AttributeName=itemId,AttributeType=S \
  --key-schema \
    AttributeName=userId,KeyType=HASH \
    AttributeName=itemId,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --region us-west-2

# 4. Set environment variables and run
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-west-2
export DYNAMODB_TABLE_NAME=ShoppingCarts
export PORT=8080

go run *.go
```

### LocalStack: Stop and Clean Up

```bash
# Stop the application (if using run-local.sh, press Ctrl+C)

# Stop and remove LocalStack container
docker stop localstack
docker rm localstack

# Or if using docker-compose
docker-compose -f docker-compose.localstack.yml down

# Remove volumes (optional - deletes all data)
docker-compose -f docker-compose.localstack.yml down -v
rm -rf localstack-data/
```

---

## AWS Deployment (Production)

Deploy the Shopping Cart API to AWS using Terraform and ECS Fargate.

### Prerequisites

- AWS Account with appropriate permissions
- AWS CLI configured (`aws configure`)
- Terraform installed
- Docker installed
- Go 1.21+ installed (for local testing)

### AWS Learner Lab Setup

If using AWS Learner Lab:

1. Start the lab and wait for AWS status to turn green
2. Click "AWS Details" → "AWS CLI" → "Show"
3. Copy the credentials and save to `~/.aws/credentials`:

```ini
[default]
aws_access_key_id=YOUR_ACCESS_KEY
aws_secret_access_key=YOUR_SECRET_KEY
aws_session_token=YOUR_SESSION_TOKEN
```

4. Set the region in `~/.aws/config`:

```ini
[default]
region=us-west-2
```

### Step 1: Set Up Infrastructure with Terraform

```bash
cd dynamodb/shopping-cart-api

# Make scripts executable
chmod +x setup-infrastructure.sh
chmod +x deploy.sh

# Run infrastructure setup
./setup-infrastructure.sh
```

This script will:

- Initialize Terraform
- Create VPC, subnets, and security groups
- Create DynamoDB table (ShoppingCarts)
- Create ECR repository for Docker images
- Create ECS cluster, task definition, and service
- Create Application Load Balancer
- Set up IAM roles and policies

**Review the planned changes** and type `yes` when prompted.

### Step 2: Deploy the Application

```bash
# Deploy the application to ECS
./deploy.sh
```

This script will:

- Build Docker image for linux/amd64 platform
- Authenticate with ECR
- Push the image to ECR
- Update ECS service with new image
- Wait for the deployment to stabilize
- Display the API endpoint URL

### Step 3: Get the API Endpoint

The deployment script will output the ALB URL. You can also retrieve it manually:

```bash
aws elbv2 describe-load-balancers \
  --names homework7-alb \
  --region us-west-2 \
  --query 'LoadBalancers[0].DNSName' \
  --output text
```

The API will be available at: `http://<ALB-DNS-NAME>`

### AWS: Update Deployment

To deploy code changes:

```bash
cd dynamodb/shopping-cart-api
./deploy.sh
```

### AWS: Clean Up Resources

```bash
cd dynamodb/terraform

# Destroy all infrastructure
terraform destroy

# Confirm by typing 'yes' when prompted
```

**Note:** This will delete:

- All DynamoDB data
- ECR repository and images
- ECS cluster and services
- Load balancer
- VPC and networking resources
- **This cannot be undone!**

---

## Testing the API

### Health Check

```bash
# LocalStack
curl http://localhost:8080/health

# AWS
curl http://<ALB-DNS-NAME>/health
```

Expected response:

```json
{
  "status": "healthy",
  "service": "shopping-cart-api"
}
```

### Create Shopping Cart

```bash
# LocalStack
curl -X POST http://localhost:8080/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 12345}'

# AWS
curl -X POST http://<ALB-DNS-NAME>/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 12345}'
```

Expected response:

```json
{
  "shopping_cart_id": "user-12345",
  "customer_id": 12345
}
```

### Add Item to Cart

```bash
# LocalStack
curl -X POST http://localhost:8080/shopping-carts/user-12345/items \
  -H "Content-Type: application/json" \
  -d '{
    "item_id": "item-789",
    "product_name": "Laptop",
    "quantity": 1,
    "price": 999.99
  }'

# AWS
curl -X POST http://<ALB-DNS-NAME>/shopping-carts/user-12345/items \
  -H "Content-Type: application/json" \
  -d '{
    "item_id": "item-789",
    "product_name": "Laptop",
    "quantity": 1,
    "price": 999.99
  }'
```

Expected response:

```json
{
  "message": "Item added successfully"
}
```

### Get Shopping Cart

```bash
# LocalStack
curl http://localhost:8080/shopping-carts/user-12345

# AWS
curl http://<ALB-DNS-NAME>/shopping-carts/user-12345
```

Expected response:

```json
{
  "shopping_cart_id": "user-12345",
  "customer_id": 12345,
  "items": [
    {
      "item_id": "item-789",
      "product_name": "Laptop",
      "quantity": 1,
      "price": 999.99
    }
  ],
  "total_price": 999.99
}
```

### Using Postman

Import the Postman collection for comprehensive testing:

**File:** `dynamodb/tests/Shopping_Cart_API_DynamoDB.postman_collection.json`

1. Open Postman
2. Click "Import" → Select the collection file
3. Update the `baseUrl` variable:
   - LocalStack: `http://localhost:8080`
   - AWS: `http://<ALB-DNS-NAME>`
4. Run the collection

---

## Troubleshooting

### LocalStack Issues

**Problem:** LocalStack container won't start

```bash
# Check Docker is running
docker info

# Check logs
docker logs localstack

# Remove and recreate
docker stop localstack && docker rm localstack
./setup-localstack.sh
```

**Problem:** DynamoDB not available

```bash
# Check LocalStack health
curl http://localhost:4566/_localstack/health

# Recreate table
./setup-localstack.sh
# Choose "yes" to recreate table when prompted
```

**Problem:** Application can't connect to LocalStack

```bash
# Verify environment variables
echo $AWS_ENDPOINT_URL  # Should be http://localhost:4566
echo $AWS_ACCESS_KEY_ID  # Should be test

# Check if LocalStack is accessible
curl http://localhost:4566/_localstack/health
```

### AWS Issues

**Problem:** Terraform fails during apply

```bash
# Check AWS credentials
aws sts get-caller-identity

# Check region
aws configure get region

# Destroy and retry
cd dynamodb/terraform
terraform destroy
terraform apply
```

**Problem:** Docker build fails for ARM Macs

```bash
# Ensure you're building for linux/amd64
docker build --platform linux/amd64 -t shopping-cart-api .
```

**Problem:** ECS task fails to start

```bash
# Check ECS task logs
aws ecs describe-tasks \
  --cluster homework8-cluster \
  --tasks $(aws ecs list-tasks --cluster homework8-cluster --service shopping-cart-api --query 'taskArns[0]' --output text) \
  --query 'tasks[0].containers[0].reason'

# Check CloudWatch logs
aws logs tail /ecs/shopping-cart-api --follow
```

**Problem:** Can't access ALB endpoint

```bash
# Verify ALB is active
aws elbv2 describe-load-balancers --names homework7-alb

# Check target health
aws elbv2 describe-target-health \
  --target-group-arn $(aws elbv2 describe-target-groups --names shopping-cart-api-tg --query 'TargetGroups[0].TargetGroupArn' --output text)

# Check security groups allow inbound traffic on port 80
```

**Problem:** AWS Learner Lab session expired

```bash
# Refresh credentials in ~/.aws/credentials
# Then re-run commands
```

### Application Issues

**Problem:** API returns 500 errors

```bash
# Check application logs
# LocalStack:
docker logs shopping-cart-api  # if using docker-compose

# AWS:
aws logs tail /ecs/shopping-cart-api --follow
```

**Problem:** Items not persisting

```bash
# Verify DynamoDB table exists
# LocalStack:
aws dynamodb list-tables --endpoint-url http://localhost:4566

# AWS:
aws dynamodb list-tables --region us-west-2

# Check table contents
# LocalStack:
aws dynamodb scan --table-name ShoppingCarts --endpoint-url http://localhost:4566

# AWS:
aws dynamodb scan --table-name ShoppingCarts --region us-west-2
```

---

## Key Differences: LocalStack vs AWS

| Feature              | LocalStack                | AWS                  |
| -------------------- | ------------------------- | -------------------- |
| **Cost**             | Free                      | Charges apply        |
| **Setup Time**       | < 1 minute                | 5-10 minutes         |
| **Endpoint**         | `http://localhost:4566`   | ALB DNS name         |
| **Credentials**      | Dummy (`test`/`test`)     | Real AWS credentials |
| **Data Persistence** | Lost when container stops | Persistent           |
| **Performance**      | Local machine             | AWS infrastructure   |
| **Best For**         | Development, testing      | Production, demos    |

---

## Environment Variables Reference

### LocalStack

```bash
AWS_ENDPOINT_URL=http://localhost:4566
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_REGION=us-west-2
DYNAMODB_TABLE_NAME=ShoppingCarts
PORT=8080
```

### AWS

```bash
# Set via ECS task definition / No endpoint URL needed
AWS_REGION=us-west-2
DYNAMODB_TABLE_NAME=ShoppingCarts
PORT=8080
```

---

## Additional Resources

- [LocalStack Documentation](https://docs.localstack.cloud/)
- [AWS DynamoDB Documentation](https://docs.aws.amazon.com/dynamodb/)
- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)

---

## Summary

- **LocalStack:** Fast local development with zero AWS costs
- **AWS:** Production-ready deployment with full AWS infrastructure
- Both approaches use the same application code
- Switch between environments by changing environment variables
