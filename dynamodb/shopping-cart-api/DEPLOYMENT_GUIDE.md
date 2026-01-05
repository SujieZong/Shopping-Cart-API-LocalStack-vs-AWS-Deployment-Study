# 🚀 Deployment and Testing Guide

Complete step-by-step guide to deploy and test the Shopping Cart API on AWS.

---

## 📋 Prerequisites

### Required Tools

- ✅ AWS CLI installed and configured
- ✅ Docker installed and running
- ✅ Terraform installed (v1.0+)
- ✅ Postman (or curl for API testing)
- ✅ AWS credentials with appropriate permissions

### AWS Permissions Required

- DynamoDB (create tables)
- ECR (create repositories, push images)
- ECS (create clusters, services, task definitions)
- EC2 (VPC, subnets, security groups, ALB)
- IAM (create roles, policies)

### Check Prerequisites

```bash
# Verify AWS CLI
aws --version
aws sts get-caller-identity

# Verify Docker
docker --version

# Verify Terraform
terraform --version
```

---

## 🏗️ Step 1: Set Up Infrastructure

### 1.1 Navigate to Project Directory

```bash
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/CS6650-HW/HW8/dynamodb
```

### 1.2 Run Infrastructure Setup

```bash
cd shopping-cart-api
chmod +x setup-infrastructure.sh
./setup-infrastructure.sh
```

**What this does:**

- Initializes Terraform
- Creates DynamoDB table `ShoppingCarts`
- Creates ECR repository `shopping-cart-api`
- Sets up VPC, subnets, security groups
- Creates ECS cluster `homework7-cluster`
- Sets up Application Load Balancer
- Configures IAM roles and policies

**Expected Output:**

```
✓ Terraform initialization complete
✓ Infrastructure validated
✓ Review plan and confirm (type 'yes')
✓ Infrastructure created successfully
```

### 1.3 Note the Outputs

Save these values from Terraform output:

```bash
terraform output
```

You'll see:

- `alb_url` - Your API endpoint
- `dynamodb_table_name` - Should be "ShoppingCarts"
- `ecr_shopping_cart_api_url` - ECR repository URL

---

## 🐳 Step 2: Build and Deploy Application

### 2.1 Make Deploy Script Executable

```bash
chmod +x deploy.sh
```

### 2.2 Run Deployment

```bash
./deploy.sh
```

**What this does:**

1. Builds Docker image from `Dockerfile`
2. Authenticates with AWS ECR
3. Tags image for ECR
4. Pushes image to ECR repository
5. Updates ECS service with new image
6. Waits for deployment to stabilize
7. Displays API endpoint

**Expected Output:**

```
=== Shopping Cart API Deployment Script ===
Step 1: Building Docker image...
Step 2: Getting ECR repository URI...
Step 3: Authenticating Docker to ECR...
Step 4: Tagging image for ECR...
Step 5: Pushing image to ECR...
Step 6: Updating ECS service...
✓ Deployment initiated successfully!
✓ Service is now stable!

=== API Endpoint ===
Base URL: http://homework7-alb-xxxxx.us-west-2.elb.amazonaws.com
Shopping Carts: http://homework7-alb-xxxxx.us-west-2.elb.amazonaws.com/shopping-carts
Health Check: http://homework7-alb-xxxxx.us-west-2.elb.amazonaws.com/health
```

### 2.3 Save Your API Endpoint

```bash
# Example (replace with your actual ALB URL)
export API_URL="http://homework7-alb-xxxxx.us-west-2.elb.amazonaws.com"
```

---

## 🧪 Step 3: Test the API

### Option A: Quick Test with curl

#### 3.1 Health Check

```bash
curl $API_URL/health
```

**Expected Response:**

```json
{
  "status": "healthy",
  "service": "shopping-cart-api"
}
```

#### 3.2 Create Shopping Cart

```bash
curl -X POST $API_URL/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 12345}'
```

**Expected Response (201 Created):**

```json
{
  "shopping_cart_id": 1698765432567
}
```

**Save the cart ID:**

```bash
export CART_ID=1698765432567  # Use the actual ID from response
```

#### 3.3 Add Item to Cart

```bash
curl -X POST $API_URL/shopping-carts/$CART_ID/items \
  -H "Content-Type: application/json" \
  -d '{"product_id": 100, "quantity": 2}'
```

**Expected Response:** `204 No Content` (empty body)

#### 3.4 Get Shopping Cart

```bash
curl $API_URL/shopping-carts/$CART_ID
```

**Expected Response (200 OK):**

```json
{
  "shopping_cart_id": 1698765432567,
  "customer_id": 12345,
  "items": [
    {
      "product_id": 100,
      "quantity": 2
    }
  ]
}
```

#### 3.5 Update Item Quantity (Override Test)

```bash
curl -X POST $API_URL/shopping-carts/$CART_ID/items \
  -H "Content-Type: application/json" \
  -d '{"product_id": 100, "quantity": 5}'

# Verify it was overridden (not added)
curl $API_URL/shopping-carts/$CART_ID
```

**Expected:** quantity should be `5` (not `7`)

---

### Option B: Test with Postman

#### 3.1 Import Collection

1. Open Postman
2. Click **Import**
3. Select file: `Shopping_Cart_API_DynamoDB.postman_collection.json`
4. Collection imported with 9 test requests

#### 3.2 Set Environment Variable

1. Click **Collections** > **Shopping Cart API - DynamoDB**
2. Click **Variables** tab
3. Update `base_url`:
   - **Current Value:** `http://your-alb-url.us-west-2.elb.amazonaws.com`
   - Use your actual ALB URL from Step 2

#### 3.3 Run Tests in Order

**Test Sequence:**

1. ✅ **Health Check** - Verify API is running
2. ✅ **Create Shopping Cart** - Creates cart, saves ID
3. ✅ **Add Item to Cart** - Adds product_id=100, quantity=2
4. ✅ **Update Item Quantity** - Updates to quantity=5 (override test)
5. ✅ **Add Another Item** - Adds product_id=101, quantity=3
6. ✅ **Get Shopping Cart** - Retrieves full cart (< 50ms test)
7. ✅ **Get Non-Existent Cart** - Tests 404 error
8. ✅ **Add Item - Invalid Data** - Tests 400 validation
9. ✅ **Create Cart - Invalid ID** - Tests 400 validation

#### 3.4 Run All Tests

- Click **Shopping Cart API - DynamoDB** collection
- Click **Run** button
- Click **Run Shopping Cart API - DynamoDB**
- All tests should pass ✅

**Expected Test Results:**

```
✓ Status code is 201
✓ Response has shopping_cart_id
✓ Response time is less than 100ms
✓ Status code is 204
✓ Response body is empty
✓ Status code is 200
✓ Response has correct structure
✓ Items array has correct data
✓ Response time is less than 50ms  ← Key requirement!
✓ Status code is 404
✓ Error response has correct format
```

---

## 🔍 Step 4: Verify in AWS Console

### 4.1 Check DynamoDB Table

1. Go to AWS Console → DynamoDB
2. Select `ShoppingCarts` table
3. Click **Explore table items**
4. You should see items with:
   - `shopping_cart_id` (Number)
   - `customer_id` (Number)
   - `items` (List)
   - `created_at`, `updated_at` (String)

### 4.2 Check ECS Service

1. Go to AWS Console → ECS
2. Select `homework7-cluster`
3. Select `shopping-cart-api` service
4. Check **Tasks** tab - should show `RUNNING`
5. Check **Metrics** tab - view request count, latency

### 4.3 Check CloudWatch Logs

1. Go to AWS Console → CloudWatch
2. Click **Log groups**
3. Select `/ecs/shopping-cart-api`
4. View logs:
   ```
   INFO: Created cart with ID: 1698765432567 for customer: 12345
   INFO: Adding new product 100 to cart 1698765432567 with quantity=2
   INFO: Retrieved cart ID: 1698765432567 with 1 items
   ```

---

## 📊 Step 5: Performance Testing

### 5.1 Test Response Time

```bash
# Run 10 requests and measure time
for i in {1..10}; do
  curl -w "\nTime: %{time_total}s\n" \
    -s -o /dev/null \
    $API_URL/shopping-carts/$CART_ID
done
```

**Expected:** All times < 0.05s (50ms requirement)

### 5.2 Create 50 Test Carts (Homework Requirement)

```bash
# Quick script to create 50 carts
for i in {0..49}; do
  CUSTOMER_ID=$((1000 + i))
  curl -X POST $API_URL/shopping-carts \
    -H "Content-Type: application/json" \
    -d "{\"customer_id\": $CUSTOMER_ID}" \
    -s | jq .
  sleep 0.1
done
```

---

## 🛑 Step 6: Cleanup (Optional)

### When you're done testing:

```bash
cd ../terraform
terraform destroy
```

**This will delete:**

- DynamoDB table
- ECR repository (and images)
- ECS cluster, services, tasks
- VPC, subnets, security groups
- Application Load Balancer
- IAM roles and policies

---

## 🐛 Troubleshooting

### Issue: "No such file or directory: deploy.sh"

```bash
chmod +x deploy.sh
```

### Issue: "Error: repository does not exist"

**Solution:** Run `setup-infrastructure.sh` first to create ECR repository

### Issue: "Health check returning 503"

**Possible causes:**

1. ECS service not running - Check ECS console
2. Task failed to start - Check CloudWatch logs
3. Security group blocking traffic - Check security group rules

### Issue: "404 on all requests"

**Solution:** Check ALB listener rules in EC2 console - ensure path `/shopping-carts*` routes to target group

### Issue: "Internal server error (500)"

**Check CloudWatch logs:**

```bash
aws logs tail /ecs/shopping-cart-api --follow
```

Common causes:

- DynamoDB table doesn't exist
- IAM permissions missing
- AWS region mismatch

### Issue: "Response time > 50ms"

**Possible causes:**

1. Cold start (first request) - retry
2. Network latency - test from EC2 in same region
3. DynamoDB throttling - check CloudWatch metrics

---

## 📝 Verification Checklist

Before submitting homework, verify:

- [ ] Infrastructure deployed successfully
- [ ] DynamoDB table `ShoppingCarts` exists
- [ ] ECS service is RUNNING
- [ ] Health check returns 200 OK
- [ ] Can create shopping cart (201 response)
- [ ] Can add items to cart (204 response)
- [ ] Can retrieve cart (200 response)
- [ ] Response time < 50ms (test with Postman)
- [ ] Quantity override works (not cumulative)
- [ ] Empty cart returns `items: []` (not null)
- [ ] Non-existent cart returns 404
- [ ] Invalid data returns 400
- [ ] CloudWatch logs show requests

---

## 🎯 Quick Reference Commands

```bash
# Deploy infrastructure
./setup-infrastructure.sh

# Deploy application
./deploy.sh

# Get ALB URL
terraform output alb_url

# Health check
curl $API_URL/health

# Create cart
curl -X POST $API_URL/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 12345}'

# Get cart
curl $API_URL/shopping-carts/{CART_ID}

# View logs
aws logs tail /ecs/shopping-cart-api --follow

# Destroy everything
terraform destroy
```

---

## 📞 Support

If you encounter issues:

1. Check CloudWatch logs: `/ecs/shopping-cart-api`
2. Verify IAM permissions for LabRole
3. Ensure DynamoDB table exists in correct region
4. Check ECS task status in console
5. Review security group rules

Good luck with your homework! 🚀
