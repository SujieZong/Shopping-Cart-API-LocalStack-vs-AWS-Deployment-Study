# Comprehensive Testing Guide: MySQL LocalStack vs AWS

This guide covers all experiments needed to compare LocalStack and AWS performance for your MySQL shopping cart API.

## 📋 Overview

You have **5 test suites** covering all required metrics:

| Test Suite               | Purpose                      | Duration | File              |
| ------------------------ | ---------------------------- | -------- | ----------------- |
| **performance.go**       | Baseline 150-op comparison   | ~5 min   | ✅ Already exists |
| **load_scaling_test.go** | Load scaling behavior        | ~15 min  | ✅ NEW - Created  |
| **cold_start_test.go**   | Cold start measurement       | ~2 min   | ✅ NEW - Created  |
| **consistency_test.go**  | Read-after-write consistency | ~5 min   | ✅ NEW - Created  |
| **failure_mode_test.go** | Error handling comparison    | ~3 min   | ✅ NEW - Created  |

---

## 🎯 Test Suite Details

### Test 1: Baseline Performance Comparison ✅ EXISTING

**File:** `performance.go`  
**What it does:** Your existing 150-operation test (50 create, 50 add, 50 get)  
**Metrics collected:**

- Average response time
- P50, P95, P99 latency
- Success rate
- Per-operation statistics

**Why keep:** ✅ Already working, provides baseline comparison

---

### Test 2: Load Scaling Behavior ⭐ NEW

**File:** `load_scaling_test.go`  
**What it does:** Progressively increase load (150 → 300 → 500 operations)  
**Metrics collected:**

- Throughput (ops/second)
- Latency degradation under load
- At what point each environment struggles
- Comparison of scaling characteristics

**Output:** `localstack_load_scaling_results.json` or `aws_load_scaling_results.json`

---

### Test 3: Cold Start Comparison ⭐ NEW

**File:** `cold_start_test.go`  
**What it does:** Measures time from infrastructure up to first successful request  
**Metrics collected:**

- Infrastructure startup time
- Time to first successful API call
- Health check latency
- Service stability

**Output:** `cold_start_localstack_results.json` or `cold_start_aws_results.json`

---

### Test 4: Consistency Testing ⭐ NEW

**File:** `consistency_test.go`  
**What it does:** Tests read-after-write consistency patterns  
**Metrics collected:**

- Create-then-read consistency
- Add-item-then-read consistency
- Concurrent update behavior
- Time to consistency

**Why important:** MySQL has different consistency characteristics than DynamoDB

**Output:** `localstack_consistency_*_results.json` or `aws_consistency_*_results.json`

---

### Test 5: Failure Mode Testing ⭐ NEW

**File:** `failure_mode_test.go`  
**What it does:** Tests error handling with invalid requests  
**Metrics collected:**

- Invalid request handling
- Missing resource behavior
- Malformed data handling
- Error response times

**Output:** `localstack_failure_*_results.json` or `aws_failure_*_results.json`

---

## 🚀 How to Run Tests

### Prerequisites

```bash
# Ensure you're in the tests directory
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/mysql/tests

# Verify Go is installed
go version

# Initialize Go module if needed
go mod init mysql-tests 2>/dev/null || true
go mod tidy
```

### 🎯 Quick Start (LocalStack)

```bash
# Terminal 1: Start LocalStack + MySQL + API
cd ..
./run-local.sh
# Leave this running (shows API logs)

# Terminal 2: Run tests
cd tests
export API_URL=http://localhost:3000

# Verify services are running
curl http://localhost:3000/health

# Run your tests (see below)
```

---

## 📊 Experiment 1: Baseline Performance Comparison

### Step 1A: Test LocalStack

```bash
# 1. Start LocalStack + MySQL + Shopping Cart API
cd ..
./run-local.sh
# This will:
# - Start LocalStack if not running
# - Start MySQL container
# - Set up the database schema
# - Start the shopping cart API on port 3000
# - Show logs in the terminal

# In a NEW terminal window, verify services are running:
# Terminal 2:
curl http://localhost:4566/_localstack/health | grep rds
# Should show: "rds":"running"

curl http://localhost:3000/health
# Should return: {"status":"healthy","database":"connected"}

# 2. Run baseline test (in Terminal 2)
cd tests
export API_URL=http://localhost:3000
go run performance.go

# Output: mysql_test_results.json
# Rename for clarity:
mv mysql_test_results.json localstack_baseline_results.json

# 3. When done, stop the API (go back to Terminal 1 and press Ctrl+C)
```

### Step 1B: Test AWS

```bash
# 1. Ensure AWS infrastructure is deployed
cd ../terraform
terraform apply

# 2. Get the ALB URL
export API_URL=http://$(terraform output -raw alb_url)
echo "Testing against: $API_URL"

# 3. Run baseline test
cd ../tests
go run performance.go

# Output: mysql_test_results.json
# Rename for clarity:
mv mysql_test_results.json aws_baseline_results.json
```

**Expected Output:**

```
=== Test Results ===
Total Operations: 150
Successful: 150
Success Rate: 100.00%

create_cart:
  Count: 50
  Avg: 45.2ms
  P50: 42.1ms | P95: 67.3ms | P99: 89.4ms

add_items:
  Count: 50
  Avg: 38.7ms
  ...
```

---

## 📊 Experiment 2: Load Scaling Test

This test progressively increases load to see how each environment handles scaling.

### Step 2A: Test LocalStack

```bash
# 1. Ensure LocalStack + API are running (from Experiment 1)
# If not, start them:
cd ..
./run-local.sh
# (Run this in a separate terminal)

# 2. In your test terminal:
cd tests
export API_URL=http://localhost:3000

# 3. Run load scaling test
go test -v -run TestLoadScaling -timeout 20m

# Output: localstack_load_scaling_results.json
```

### Step 2B: Test AWS

```bash
# 1. Ensure AWS infrastructure is deployed
cd ../terraform
terraform apply

# 2. Run load scaling test (automatically gets ALB URL)
cd ../tests
go test -v -run TestLoadScaling -timeout 20m

# Output: aws_load_scaling_results.json
```

**Expected Output:**

```
╔════════════════════════════════════════════════════════════════╗
║  Testing Load Level: 50 carts (150 total operations)          ║
╚════════════════════════════════════════════════════════════════╝

=== Load Test Summary (Load Level: 50) ===
Total Operations: 150
Duration: 8.45 seconds
Throughput: 17.75 ops/sec
Success Rate: 100.00%
Average Response Time: 56.32 ms
P50: 45.23 ms | P95: 89.45 ms | P99: 123.67 ms
```

---

## 📊 Experiment 3: Cold Start Test

This test measures how quickly each environment becomes operational from a cold start.

### Step 3A: Test LocalStack

```bash
# 1. Stop all services
cd ..
docker-compose down

# 2. Start services and immediately begin testing
./run-local.sh &
# Wait a moment for services to initialize

# 3. Run cold start test
cd tests
export API_URL=http://localhost:3000
go test -v -run TestColdStart -timeout 10m

# Output: cold_start_localstack_results.json
```

### Step 3B: Test AWS

```bash
# 1. Destroy infrastructure (optional, for true cold start)
cd ../terraform
terraform destroy -auto-approve

# 2. Deploy fresh infrastructure
terraform apply -auto-approve

# 3. Run cold start test (it will wait for service to be ready)
cd ../tests
go test -v -run TestColdStart -timeout 10m

# Output: cold_start_aws_results.json
```

**Expected Output:**

```
╔════════════════════════════════════════╗
║      COLD START MEASUREMENT TEST      ║
╚════════════════════════════════════════╝

Waiting for service to become ready...
Attempt 1/60...
Attempt 2/60...
✓ Service ready after 3 attempts (15.23 seconds)

╔════════════════════════════════════════════════════════════════╗
║                  COLD START SUMMARY                           ║
╚════════════════════════════════════════════════════════════════╝

Environment: localstack
Infrastructure Up Time: 15.23 seconds
Time to First Successful Request: 0.45 seconds
Total Cold Start Time: 15.68 seconds
```

---

## 📊 Experiment 4: Consistency Testing

This test examines read-after-write consistency behavior.

### Step 4A: Test LocalStack

```bash
# 1. Ensure LocalStack + API are running
cd ..
./run-local.sh
# (Run this in a separate terminal if not already running)

# 2. Run consistency tests
cd tests
export API_URL=http://localhost:3000

# Run all consistency tests
go test -v -run TestCreateThenReadConsistency -timeout 5m
go test -v -run TestAddItemThenReadConsistency -timeout 5m
go test -v -run TestRapidConcurrentUpdates -timeout 5m
go test -v -run TestWriteReadWritePattern -timeout 5m

# Outputs:
# - localstack_consistency_create_then_read_results.json
# - localstack_consistency_add_item_then_read_results.json
```

### Step 4B: Test AWS

```bash
# 1. Ensure AWS infrastructure is deployed
cd ../terraform
terraform apply

# 2. Run consistency tests (automatically gets ALB URL)
cd ../tests

go test -v -run TestCreateThenReadConsistency -timeout 5m
go test -v -run TestAddItemThenReadConsistency -timeout 5m
go test -v -run TestRapidConcurrentUpdates -timeout 5m
go test -v -run TestWriteReadWritePattern -timeout 5m

# Outputs:
# - aws_consistency_create_then_read_results.json
# - aws_consistency_add_item_then_read_results.json
```

**Expected Output:**

```
Running Create-then-Read Consistency Test...
✓ Cart 10000 consistent (took 45ms)
✓ Cart 10001 consistent (took 38ms)
...

=== Consistency Metrics Report ===
Total Attempts: 20
Consistent Reads: 20 (100.00%)
Inconsistent Reads: 0 (0.00%)
Failed Reads: 0

Time to Consistency:
  Min: 25ms
  Max: 67ms
  Avg: 42ms
==================================
```

---

## 📊 Experiment 5: Failure Mode Testing

This test validates error handling for invalid requests.

### Step 5A: Test LocalStack

```bash
# 1. Ensure LocalStack + API are running
cd ..
./run-local.sh
# (Run this in a separate terminal if not already running)

# 2. Run failure mode tests
cd tests
export API_URL=http://localhost:3000

# Run all failure mode tests
go test -v -run TestInvalidRequests -timeout 5m
go test -v -run TestMissingResources -timeout 5m
go test -v -run TestMalformedData -timeout 5m

# Outputs:
# - localstack_failure_invalid_requests_results.json
# - localstack_failure_missing_resources_results.json
# - localstack_failure_malformed_data_results.json
```

### Step 5B: Test AWS

```bash
# 1. Ensure AWS infrastructure is deployed
cd ../terraform
terraform apply

# 2. Run failure mode tests (automatically gets ALB URL)
cd ../tests

go test -v -run TestInvalidRequests -timeout 5m
go test -v -run TestMissingResources -timeout 5m
go test -v -run TestMalformedData -timeout 5m

# Outputs:
# - aws_failure_invalid_requests_results.json
# - aws_failure_missing_resources_results.json
# - aws_failure_malformed_data_results.json
```

**Expected Output:**

```
╔════════════════════════════════════════╗
║      FAILURE MODE: INVALID REQUESTS    ║
╚════════════════════════════════════════╝

✓ Test: missing_customer_id
  Expected Status: 400, Got: 400
  Response Time: 12.34 ms
  Handled correctly: true

✓ Test: invalid_customer_id_type
  Expected Status: 400, Got: 400
  Response Time: 11.56 ms
  Handled correctly: true
...

═══════════════════════════════════════════════════════
Summary for invalid_requests:
  Total Tests: 6
  Correctly Handled: 6 (100.00%)
  Results saved to: localstack_failure_invalid_requests_results.json
═══════════════════════════════════════════════════════
```

---

## 🔄 Running All Tests in Sequence

### LocalStack - Complete Test Suite

```bash
# Terminal 1: Start services
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/mysql
./run-local.sh

# Terminal 2: Run all tests
cd tests
export API_URL=http://localhost:3000

echo "=== Running Baseline Performance Test ==="
go run performance.go
mv mysql_test_results.json localstack_baseline_results.json

echo "=== Running Load Scaling Test ==="
go test -v -run TestLoadScaling -timeout 20m

echo "=== Running Cold Start Test ==="
go test -v -run TestColdStart -timeout 10m

echo "=== Running Consistency Tests ==="
go test -v -run TestCreateThenReadConsistency -timeout 5m
go test -v -run TestAddItemThenReadConsistency -timeout 5m
go test -v -run TestRapidConcurrentUpdates -timeout 5m
go test -v -run TestWriteReadWritePattern -timeout 5m

echo "=== Running Failure Mode Tests ==="
go test -v -run TestInvalidRequests -timeout 5m
go test -v -run TestMissingResources -timeout 5m
go test -v -run TestMalformedData -timeout 5m

echo "=== All LocalStack Tests Complete ==="
ls -lh localstack_*.json aws_*.json 2>/dev/null
```

### AWS - Complete Test Suite

```bash
# Ensure AWS infrastructure is deployed
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/mysql/terraform
terraform apply -auto-approve

cd ../tests

echo "=== Running Baseline Performance Test ==="
go run performance.go
mv mysql_test_results.json aws_baseline_results.json

echo "=== Running Load Scaling Test ==="
go test -v -run TestLoadScaling -timeout 20m

echo "=== Running Cold Start Test ==="
go test -v -run TestColdStart -timeout 10m

echo "=== Running Consistency Tests ==="
go test -v -run TestCreateThenReadConsistency -timeout 5m
go test -v -run TestAddItemThenReadConsistency -timeout 5m
go test -v -run TestRapidConcurrentUpdates -timeout 5m
go test -v -run TestWriteReadWritePattern -timeout 5m

echo "=== Running Failure Mode Tests ==="
go test -v -run TestInvalidRequests -timeout 5m
go test -v -run TestMissingResources -timeout 5m
go test -v -run TestMalformedData -timeout 5m

echo "=== All AWS Tests Complete ==="
ls -lh aws_*.json
```

---

## 📈 Comparing Results

After running all tests, you'll have JSON files for comparison:

### LocalStack Results

- `localstack_baseline_results.json`
- `localstack_load_scaling_results.json`
- `cold_start_localstack_results.json`
- `localstack_consistency_*_results.json`
- `localstack_failure_*_results.json`

### AWS Results

- `aws_baseline_results.json`
- `aws_load_scaling_results.json`
- `cold_start_aws_results.json`
- `aws_consistency_*_results.json`
- `aws_failure_*_results.json`

### Key Metrics to Compare

1. **Performance**: Average response time, P95, P99
2. **Scalability**: Throughput degradation under load
3. **Cold Start**: Time to first successful request
4. **Consistency**: % of immediate consistent reads
5. **Reliability**: Error handling correctness

---

## 🐛 Troubleshooting

### LocalStack Issues

```bash
# Check LocalStack status
curl http://localhost:4566/_localstack/health

# Check MySQL container
docker ps | grep mysql

# Check API logs
cd ..
docker-compose logs -f shopping-cart-api

# Restart everything
docker-compose down
./run-local.sh
```

### AWS Issues

```bash
# Check Terraform outputs
cd ../terraform
terraform output

# Check ECS task status
aws ecs list-tasks --cluster shopping-cart-cluster
aws ecs describe-tasks --cluster shopping-cart-cluster --tasks <task-arn>

# Check RDS status
aws rds describe-db-instances --db-instance-identifier shopping-cart-mysql
```

### Test Failures

```bash
# Increase timeouts if needed
go test -v -run TestName -timeout 30m

# Run with more verbose output
go test -v -run TestName 2>&1 | tee test_output.log

# Check connectivity
curl -v http://localhost:3000/health
```

---

## 📝 Notes

- **API URL**: LocalStack uses `http://localhost:3000`, AWS uses ALB URL from Terraform
- **Concurrency**: Tests use controlled concurrency to avoid overwhelming the API
- **Retries**: Consistency tests include retry logic for eventual consistency
- **Timeouts**: Set appropriate timeouts based on expected operation duration
- **Environment Detection**: Tests automatically detect environment (LocalStack vs AWS) based on URL

---

## ✅ Checklist

Before comparing MySQL and DynamoDB:

- [ ] Run all LocalStack tests for MySQL
- [ ] Run all AWS tests for MySQL
- [ ] Run all LocalStack tests for DynamoDB
- [ ] Run all AWS tests for DynamoDB
- [ ] Collect all JSON result files
- [ ] Compare performance metrics
- [ ] Document findings in your report

---

## 🎓 Key Differences from DynamoDB Tests

1. **No Flash Sale Test**: Excluded as requested
2. **Consistency Model**: MySQL has strong consistency by default (different from DynamoDB's eventual consistency)
3. **Port Numbers**: MySQL API uses port 3000 (DynamoDB uses 8080)
4. **Connection Handling**: MySQL uses connection pooling (different characteristics)
5. **Error Messages**: MySQL error responses may differ from DynamoDB

Good luck with your testing! 🚀
