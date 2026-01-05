# Shopping Cart API - MySQL Implementation

## Deployment Guide

This guide covers both **local development** and **AWS production deployment** for the MySQL-based Shopping Cart API.

---

## 🎉 Current Deployment Status

**✅ LocalStack + RDS Deployment: ACTIVE**

All services are running successfully:

- **LocalStack Pro**: Running (v4.11.2.dev5) on port 4566
- **RDS MySQL**: Available (8.0.40) - Instance ID: `shopping-cart-mysql`
- **MySQL Container**: Running (ls-mysql-046d0024) - Database: `shopping_cart_db`
- **Shopping Cart API**: Running on port 3000 - Status: Healthy

**Quick Health Checks:**

```bash
# API Health (Expected: {"service":"shopping-cart-mysql","status":"healthy"})
curl http://localhost:3000/health

# LocalStack RDS Status
awslocal rds describe-db-instances --db-instance-identifier shopping-cart-mysql

# Test API Endpoint
curl http://localhost:3000/shopping-carts/1
```

**Database Information:**

- Tables: `shopping_carts` (2 rows), `cart_items` (1 row)
- Credentials: root/MySecurePass123!, admin/MySecurePass123!
- Port: 4510 (LocalStack RDS endpoint)

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development](#local-development)
3. [AWS Production Deployment](#aws-production-deployment)
4. [Testing](#testing)
5. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### For Local Development

- **Docker Desktop** installed and running
- **Docker Compose** (included with Docker Desktop)
- Basic command line knowledge

### For AWS Deployment

- **AWS Account** with appropriate permissions
- **AWS CLI** configured with credentials
- **Terraform** (>= 1.0)
- **Docker** installed
- Access to AWS services: ECS, RDS, ECR, ALB, VPC

---

## Local Development

### Quick Start (LocalStack + RDS)

**New Approach:** Using LocalStack Pro to simulate AWS RDS locally

The fastest way to get started locally:

```bash
cd mysql

# Set your LocalStack Pro auth token (required)
export LOCALSTACK_AUTH_TOKEN=your-token-here

# Run the setup (one command does it all)
./run-local.sh
```

This script will:

1. ✅ Check if Docker is running
2. ✅ Verify LocalStack Pro token is set
3. 🚀 Start LocalStack with RDS support (if not running)
4. 📊 Create RDS MySQL instance in LocalStack
5. 🐳 Build and start API container
6. ⏳ Wait for services to be ready
7. 🧪 Run basic API tests
8. 📊 Display connection information

**Why LocalStack?**

- ✅ Simulates real AWS RDS environment locally
- ✅ Tests infrastructure code before AWS deployment
- ✅ LocalStack Pro supports RDS with MySQL
- ✅ Same experience as production without AWS costs

### Manual Setup (LocalStack)

If you prefer to run commands manually:

#### Step 1: Set LocalStack Auth Token

```bash
export LOCALSTACK_AUTH_TOKEN=your-token-here
```

#### Step 2: Initialize LocalStack and RDS

```bash
cd mysql
chmod +x setup-localstack.sh
./setup-localstack.sh
```

This will:

- Start LocalStack Pro container
- Create RDS MySQL instance
- Initialize database schema

#### Step 3: Start API Service

```bash
docker-compose -f docker-compose.localstack.yml up -d api
```

#### Step 4: Check Service Health

```bash
# Check LocalStack
curl http://localhost:4566/_localstack/health

# Check RDS status
awslocal rds describe-db-instances --db-instance-identifier shopping-cart-mysql

# Check API
curl http://localhost:3000/health
```

#### Step 5: Test the API

```bash
# Create a shopping cart
curl -X POST http://localhost:3000/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 123}'

# Add item to cart (replace {cart_id} with actual ID)
curl -X POST http://localhost:3000/shopping-carts/{cart_id}/items \
  -H "Content-Type: application/json" \
  -d '{"product_id": 456, "quantity": 2}'

# Get cart
curl http://localhost:3000/shopping-carts/{cart_id}
```

### Local Configuration

**Service URLs:**

- **API**: `http://localhost:3000`
- **Health Check**: `http://localhost:3000/health`
- **LocalStack**: `http://localhost:4566`
- **LocalStack Health**: `http://localhost:4566/_localstack/health`

**API Endpoints (Important!):**

```bash
# Health Check
curl http://localhost:3000/health

# Get Shopping Cart (use cart ID)
curl http://localhost:3000/shopping-carts/{cart_id}

# Create Shopping Cart
curl -X POST http://localhost:3000/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 123}'

# Add Item to Cart
curl -X POST http://localhost:3000/shopping-carts/{cart_id}/items \
  -H "Content-Type: application/json" \
  -d '{"product_id": 456, "quantity": 2}'

# Delete Shopping Cart
curl -X DELETE http://localhost:3000/shopping-carts/{cart_id}
```

**RDS Connection (via LocalStack):**

- **RDS Instance ID**: `shopping-cart-mysql`
- **MySQL Container**: `ls-mysql-046d0024`
- **Database**: `shopping_cart_db`
- **Root User**: `root` / `MySecurePass123!`
- **Admin User**: `admin` / `MySecurePass123!`
- **LocalStack RDS Port**: `4510`
- **MySQL Direct Port**: `3306` (internal to container)
- **Endpoint**: `localhost.localstack.cloud:4510`

**Database Tables:**

- `shopping_carts` - Stores shopping cart metadata
- `cart_items` - Stores items in each cart

**Connect to MySQL:**

```bash
# Option 1: Connect to MySQL Container Directly (Recommended)
docker exec -it ls-mysql-046d0024 mysql -uroot -p'MySecurePass123!' shopping_cart_db

# Option 2: Via LocalStack RDS Endpoint
RDS_ENDPOINT=$(awslocal rds describe-db-instances \
  --db-instance-identifier shopping-cart-mysql \
  --query 'DBInstances[0].Endpoint.Address' \
  --output text)

echo "RDS Endpoint: ${RDS_ENDPOINT}"

# Option 3: Check Database Content
docker exec ls-mysql-046d0024 mysql -uroot -p'MySecurePass123!' shopping_cart_db \
  -e "SHOW TABLES; SELECT COUNT(*) FROM shopping_carts; SELECT COUNT(*) FROM cart_items;"
```

**Useful LocalStack Commands:**

```bash
# Check RDS instance details
awslocal rds describe-db-instances --db-instance-identifier shopping-cart-mysql

# Check LocalStack services status
curl -s http://localhost:4566/_localstack/health | python3 -m json.tool

# View running containers
docker ps | grep -E "localstack|mysql|shopping-cart"

# View API logs
docker logs shopping-cart-api -f

# View MySQL logs
docker logs ls-mysql-046d0024 --tail 50

# View LocalStack logs
docker logs shopping-cart-localstack -f

# Stop everything
docker-compose -f docker-compose.localstack.yml down

# Clean restart (removes all data)
docker-compose -f docker-compose.localstack.yml down -v
./run-local.sh
```

**MySQL Connection:**

- Host: `localhost` (or `ls-mysql-046d0024` container name)
- Port: `3306` (internal), `4510` (LocalStack RDS)
- Database: `shopping_cart_db`
- Root User: `root` / `MySecurePass123!`
- Admin User: `admin` / `MySecurePass123!`

### Verify Your Deployment

```bash
# 1. Check all containers are running
docker ps | grep -E "localstack|mysql|shopping-cart"

# 2. Verify API is healthy
curl http://localhost:3000/health
# Expected: {"service":"shopping-cart-mysql","status":"healthy"}

# 3. Verify LocalStack RDS
awslocal rds describe-db-instances --db-instance-identifier shopping-cart-mysql
# Expected: "DBInstanceStatus": "available"

# 4. Verify MySQL database
docker exec ls-mysql-046d0024 mysql -uroot -p'MySecurePass123!' shopping_cart_db \
  -e "SHOW TABLES;"
# Expected: shopping_carts, cart_items

# 5. Test API functionality
curl http://localhost:3000/shopping-carts/1
# Expected: Cart data in JSON format
```

### Useful Local Commands

```bash
# View all logs
docker-compose -f docker-compose.localstack.yml logs -f

# View API logs only
docker logs shopping-cart-api -f

# View MySQL logs only
docker-compose logs -f mysql

# Connect to MySQL CLI
docker exec -it shopping-cart-mysql mysql -u admin -pMySecurePass123! shopping_cart_db

# Stop services
docker-compose down

# Stop and remove all data
docker-compose down -v

# Rebuild and restart
docker-compose up -d --build
```

### Reinitialize Database

If you need to reset the database:

```bash
./setup-local-db.sh
```

Or manually:

```bash
docker exec -i shopping-cart-mysql mysql -u admin -pMySecurePass123! shopping_cart_db < src/db/setup.sql
```

---

## AWS Production Deployment

### Overview

The Terraform configuration deploys:

- **VPC** with public and private subnets across 2 AZs
- **RDS MySQL** (db.t3.micro) in private subnets
- **ECS Fargate** cluster with 2-4 tasks
- **Application Load Balancer** for traffic distribution
- **ECR** for container image storage
- **Auto-scaling** based on CPU utilization

### Prerequisites Check

```bash
# Verify AWS CLI is configured
aws sts get-caller-identity

# Verify Terraform is installed
terraform --version

# Verify Docker is running
docker info
```

### Deployment Steps

#### Option 1: Automated Deployment (Recommended)

Use the provided deployment script:

```bash
cd mysql
./deploy.sh
```

This script will:

1. 📦 Deploy infrastructure with Terraform
2. 🐳 Build Docker image
3. 🔐 Login to AWS ECR
4. ⬆️ Push image to ECR
5. ♻️ Update ECS service
6. ⏳ Wait for deployment to complete
7. 🧪 Run health check tests
8. 📊 Display endpoint URLs

#### Option 2: Manual Deployment

##### Step 1: Deploy Infrastructure

```bash
cd mysql/terraform
terraform init
terraform plan
terraform apply -auto-approve
```

Save the outputs:

```bash
terraform output alb_url
terraform output rds_endpoint
terraform output ecr_repository_url
```

##### Step 2: Build and Push Docker Image

```bash
cd ../src

# Get AWS account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION="us-west-2"
ECR_REPO="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/hw8-mysql-app"

# Build image
docker build --platform linux/amd64 -t hw8-mysql-app .

# Login to ECR
aws ecr get-login-password --region ${AWS_REGION} | \
    docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Tag and push
docker tag hw8-mysql-app:latest ${ECR_REPO}:latest
docker push ${ECR_REPO}:latest
```

##### Step 3: Update ECS Service

```bash
cd ../terraform

ECS_CLUSTER=$(terraform output -raw ecs_cluster_name)
ECS_SERVICE=$(terraform output -raw ecs_service_name)

aws ecs update-service \
    --cluster ${ECS_CLUSTER} \
    --service ${ECS_SERVICE} \
    --force-new-deployment \
    --region us-west-2
```

##### Step 4: Wait for Deployment

```bash
aws ecs wait services-stable \
    --cluster ${ECS_CLUSTER} \
    --services ${ECS_SERVICE} \
    --region us-west-2
```

##### Step 5: Test the Deployment

```bash
ALB_URL=$(terraform output -raw alb_url)

# Health check
curl ${ALB_URL}/health

# Create cart
curl -X POST ${ALB_URL}/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 123}'
```

### AWS Configuration

**Default Settings (can be modified in terraform/main.tf):**

```hcl
variable "aws_region" {
  default = "us-west-2"
}

variable "project_name" {
  default = "hw8-mysql"
}

variable "db_username" {
  default = "admin"
}

variable "db_password" {
  default = "MySecurePass123!"
  sensitive = true
}
```

**ECS Task Configuration:**

- CPU: 256 (0.25 vCPU)
- Memory: 1024 MB (1 GB)
- Desired Count: 2-4 (auto-scaling enabled)
- Platform: Fargate

**RDS Configuration:**

- Instance Class: db.t3.micro
- Engine: MySQL 8.0
- Storage: 20 GB GP2
- Multi-AZ: No (can be enabled for production)

### Monitoring and Logs

#### View ECS Logs

```bash
# Using AWS Console
# Navigate to: CloudWatch > Log Groups > /ecs/hw8-mysql

# Using AWS CLI
aws logs tail /ecs/hw8-mysql --follow --region us-west-2
```

#### View RDS Metrics

```bash
# Navigate to AWS Console > RDS > Databases > hw8-mysql-mysql
# Check: Monitoring tab for CPU, connections, IOPS
```

#### Check ECS Service Status

```bash
cd mysql/terraform
ECS_CLUSTER=$(terraform output -raw ecs_cluster_name)
ECS_SERVICE=$(terraform output -raw ecs_service_name)

aws ecs describe-services \
    --cluster ${ECS_CLUSTER} \
    --services ${ECS_SERVICE} \
    --region us-west-2
```

### Scaling Configuration

The deployment includes auto-scaling:

- **Min Tasks**: 2
- **Max Tasks**: 4
- **Scale-up Trigger**: CPU > 75%
- **Scale-down Trigger**: CPU < 75%

To manually scale:

```bash
aws ecs update-service \
    --cluster ${ECS_CLUSTER} \
    --service ${ECS_SERVICE} \
    --desired-count 3 \
    --region us-west-2
```

### Cleanup AWS Resources

**⚠️ Warning: This will delete all resources and data!**

```bash
cd mysql/terraform
terraform destroy -auto-approve
```

Or use the cleanup script:

```bash
cd mysql
./cleanup.sh
```

---

## Testing

### Local Testing

Run the provided test script:

```bash
cd mysql
./test.sh
```

Or use the performance test:

```bash
cd mysql/tests
go run performance.go
```

### AWS Testing

```bash
# Get ALB URL from Terraform
cd mysql/terraform
ALB_URL=$(terraform output -raw alb_url)

# Run tests against AWS endpoint
export API_URL=${ALB_URL}
cd ../tests
go run performance.go
```

### API Endpoints

1. **Health Check**

   ```bash
   GET /health
   ```

2. **Create Shopping Cart**

   ```bash
   POST /shopping-carts
   Body: {"customer_id": 123}
   Response: {"shopping_cart_id": 1}
   ```

3. **Add Item to Cart**

   ```bash
   POST /shopping-carts/:id/items
   Body: {"product_id": 456, "quantity": 2}
   Response: 204 No Content
   ```

4. **Get Shopping Cart**
   ```bash
   GET /shopping-carts/:id
   Response: {
     "shopping_cart_id": 1,
     "customer_id": 123,
     "items": [
       {"product_id": 456, "quantity": 2}
     ]
   }
   ```

---

## Troubleshooting

### Local Issues

#### Issue: Docker containers won't start

**Solution:**

```bash
# Check Docker is running
docker info

# Remove old containers
docker-compose down -v

# Rebuild from scratch
docker-compose up -d --build
```

#### Issue: MySQL connection refused

**Solution:**

```bash
# Check MySQL logs
docker-compose logs mysql

# Wait longer for MySQL to initialize (first start takes 30-60 seconds)
docker exec shopping-cart-mysql mysqladmin ping -h localhost -u admin -pMySecurePass123!
```

#### Issue: API can't connect to MySQL

**Solution:**

```bash
# Check if both containers are in the same network
docker network ls
docker network inspect mysql_shopping-cart-network

# Restart API container
docker-compose restart api
```

#### Issue: Port 3306 already in use

**Solution:**

```bash
# If you have MySQL installed locally, stop it or change the port in docker-compose.yml
# Change ports: "3307:3306" instead of "3306:3306"

# On macOS:
brew services stop mysql

# Or edit docker-compose.yml and use port 3307
```

### AWS Issues

#### Issue: Terraform fails to apply

**Solution:**

```bash
# Check AWS credentials
aws sts get-caller-identity

# Check for resource limits
aws service-quotas list-service-quotas --service-code ec2

# Destroy and retry
terraform destroy -auto-approve
terraform apply -auto-approve
```

#### Issue: ECS tasks fail to start

**Solution:**

```bash
# Check ECS logs
aws logs tail /ecs/hw8-mysql --follow --region us-west-2

# Common causes:
# 1. Image not found in ECR - rebuild and push
# 2. RDS not ready - wait 2-3 minutes after terraform apply
# 3. Security group issues - check terraform/main.tf
```

#### Issue: Health checks failing

**Solution:**

```bash
# Check if API can reach RDS
# 1. Verify security groups allow ECS -> RDS on port 3306
# 2. Check RDS is in "Available" state
aws rds describe-db-instances --db-instance-identifier hw8-mysql-mysql --region us-west-2

# 3. Check environment variables in ECS task definition
aws ecs describe-task-definition --task-definition hw8-mysql-app --region us-west-2
```

#### Issue: ALB returns 503 Service Unavailable

**Solution:**

```bash
# Check target health
aws elbv2 describe-target-health \
    --target-group-arn $(terraform output -raw target_group_arn) \
    --region us-west-2

# Common causes:
# 1. No healthy targets - check ECS tasks
# 2. Health check path incorrect - verify /health endpoint works
# 3. Security group blocking traffic - check ECS tasks security group
```

#### Issue: Can't push to ECR

**Solution:**

```bash
# Re-login to ECR
aws ecr get-login-password --region us-west-2 | \
    docker login --username AWS --password-stdin $(aws sts get-caller-identity --query Account --output text).dkr.ecr.us-west-2.amazonaws.com

# Verify repository exists
aws ecr describe-repositories --repository-names hw8-mysql-app --region us-west-2
```

#### Issue: RDS connection timeout

**Solution:**

```bash
# 1. Verify RDS security group allows inbound from ECS security group
# 2. Check RDS is in private subnet
# 3. Verify NAT Gateway is working (for ECS to reach internet for ECR)
# 4. Check CloudWatch logs for specific error messages

# Test from ECS task (get task ID from console)
aws ecs execute-command \
    --cluster hw8-mysql-cluster \
    --task TASK_ID \
    --container shopping-cart-api \
    --interactive \
    --command "/bin/sh"
```

### Database Issues

#### Issue: Database schema not initialized

**Solution (Local):**

```bash
./setup-local-db.sh
```

**Solution (AWS):**

```bash
# The schema is automatically initialized by the application on startup
# Check ECS logs to verify initialization
aws logs tail /ecs/hw8-mysql --follow --region us-west-2 | grep "schema"
```

#### Issue: Data inconsistency

**Solution:**

```bash
# Local: Reset database
docker-compose down -v
docker-compose up -d

# AWS: Connect to RDS and run cleanup
# (Not recommended for production - use proper migration tools)
```

---

## Architecture Overview

### Local Architecture

```
┌─────────────────┐
│   Docker Host   │
│                 │
│  ┌───────────┐  │
│  │    API    │  │  Port 3000
│  │ Container │◄─┼─────────── HTTP Requests
│  └─────┬─────┘  │
│        │        │
│        │ TCP    │
│        ▼        │
│  ┌───────────┐  │
│  │   MySQL   │  │  Port 3306
│  │ Container │  │
│  └───────────┘  │
│                 │
│   Bridge        │
│   Network       │
└─────────────────┘
```

### AWS Architecture

```
                        Internet
                           │
                           ▼
                    ┌─────────────┐
                    │     ALB     │
                    │  (Public)   │
                    └──────┬──────┘
                           │
              ┌────────────┴────────────┐
              │                         │
         ┌────▼────┐              ┌────▼────┐
         │   ECS   │              │   ECS   │
         │  Task   │              │  Task   │
         │   (AZ1) │              │   (AZ2) │
         └────┬────┘              └────┬────┘
              │                         │
              └────────────┬────────────┘
                           │
                     ┌─────▼─────┐
                     │    RDS    │
                     │   MySQL   │
                     │ (Private) │
                     └───────────┘
```

---

## Additional Resources

- **Terraform Documentation**: https://registry.terraform.io/providers/hashicorp/aws/latest/docs
- **AWS ECS Best Practices**: https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/
- **MySQL 8.0 Documentation**: https://dev.mysql.com/doc/refman/8.0/en/
- **Docker Compose**: https://docs.docker.com/compose/

---

## Support

For issues or questions:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review CloudWatch logs for AWS deployments
3. Check Docker logs for local deployments
4. Verify all prerequisites are met

---

**Last Updated**: November 29, 2025
