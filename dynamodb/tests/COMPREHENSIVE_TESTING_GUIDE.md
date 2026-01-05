# Comprehensive Testing Guide: LocalStack vs AWS

This guide covers all experiments needed to compare LocalStack and AWS performance for your DynamoDB shopping cart API.

## 📋 Overview

You have **6 test suites** covering all required metrics:

| Test Suite               | Purpose                       | Duration | Locust?                      |
| ------------------------ | ----------------------------- | -------- | ---------------------------- |
| **performance_test.go**  | Baseline 150-op comparison    | ~5 min   | ❌ No - Go is sufficient     |
| **load_scaling_test.go** | Load scaling behavior         | ~15 min  | ❌ No - built-in concurrency |
| **flash_sale_test.go**   | 🔥 Flash sale simulation      | 1 min    | ❌ No - 500 concurrent users |
| **cold_start_test.go**   | Cold start measurement        | ~2 min   | ❌ No - timing focused       |
| **consistency_test.go**  | Eventual consistency patterns | ~5 min   | ❌ No - precision needed     |
| **failure_mode_test.go** | Error handling comparison     | ~3 min   | ❌ No - specific scenarios   |

### ❌ Why NOT Use Locust?

**You don't need Locust because:**

1. ✅ **Go's concurrency is built-in** - Your tests already use goroutines for parallel requests
2. ✅ **Precise measurements needed** - Go gives you microsecond precision
3. ✅ **Small to medium scale** - Testing 150-500 operations, not 10,000+
4. ✅ **Environment comparison** - Need identical test logic for fair comparison
5. ✅ **JSON output** - Go tests output structured JSON for easy analysis

**When you WOULD use Locust:**

- Testing 10,000+ concurrent users
- Simulating realistic user behavior patterns
- Distributed load testing across multiple machines
- Web UI for real-time monitoring

---

## 🎯 Test Suite Details

### Test 1: Baseline Performance Comparison ✅ KEEP THIS

**File:** `performance_test.go`  
**What it does:** Your existing 150-operation test (50 create, 50 add, 50 get)  
**Metrics collected:**

- Average response time
- P50, P95, P99 latency
- Success rate
- Per-operation statistics

**Why keep:** ✅ Already working, provides baseline comparison

---

### Test 2: Load Scaling Behavior ⭐ NEW - ADDED

**File:** `load_scaling_test.go`  
**What it does:** Progressively increase load (150 → 300 → 500 operations)  
**Metrics collected:**

- Throughput (ops/second)
- Latency degradation under load
- At what point each environment struggles
- Comparison of scaling characteristics

**Output:** `load_scaling_results.json`

---

### Test 3: Flash Sale Load Test 🔥 NEW - HIGH CONCURRENCY

**File:** `flash_sale_test.go`  
**What it does:** Simulates a realistic flash sale with 500 concurrent users for 1 minute  
**Key features:**

- ✅ 500 concurrent users
- ✅ 1 minute duration
- ✅ Headless (no UI needed)
- ✅ Automatic result saving to JSON
- ✅ Works with both LocalStack and AWS
- ✅ Real-time progress monitoring (5-second intervals)

**User behavior simulation:**

- Each user continuously creates carts
- Adds 1-3 random items per cart
- Retrieves cart information
- Random delays between operations (50-200ms)

**Metrics collected:**

- Total throughput (operations per second)
- Success rate (%)
- Response time percentiles (P50, P90, P95, P99, Min, Max)
- Per-operation statistics (CREATE_CART, ADD_ITEM, GET_CART)
- Time-series data (5-second windows)
- Active user count

**Output:** `flash_sale_localstack_results.json` or `flash_sale_aws_results.json`

---

### Test 4: Cold Start Comparison ⭐ NEW - ADDED

**File:** `cold_start_test.go`  
**What it does:** Measures time from infrastructure up to first successful request  
**Metrics collected:**

- Infrastructure startup time
- Time to first successful API call
- Health check latency
- Service stability

**Output:** `cold_start_aws_results.json` or `cold_start_localstack_results.json`

---

### Test 5: Consistency Testing ✅ KEEP THIS

**File:** `consistency_test.go`  
**What it does:** Tests eventual consistency patterns  
**Metrics collected:**

- Create-then-read consistency
- Add-item-then-read consistency
- Concurrent update behavior
- Time to consistency

**Why keep:** ✅ Unique to distributed systems, important for comparison

---

### Test 6: Failure Mode Testing ⭐ NEW - ADDED

**File:** `failure_mode_test.go`  
**What it does:** Tests error handling with invalid requests  
**Metrics collected:**

- Invalid request handling
- Missing resource behavior
- Malformed data handling
- Error response times

**Output:** `failure_mode_*_results.json`

---

## 🚀 How to Run Tests

### Prerequisites

```bash
# Ensure you're in the tests directory
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/dynamodb/tests

# Verify Go modules are up to date
go mod tidy
```

### 🎯 Quick Start (LocalStack)

```bash
# Terminal 1: Start LocalStack + API
cd ../shopping-cart-api
./run-local.sh
# Leave this running (shows API logs)

# Terminal 2: Run tests
cd ../tests
export BASE_URL=http://localhost:8080

# Verify services are running
curl http://localhost:8080/health

# Run your tests
go test -v -run TestDynamoDBPerformance -timeout 10m

# When done, go to Terminal 1 and press Ctrl+C
```

---

## 📊 Experiment 1: Baseline Performance Comparison

### Step 1A: Test LocalStack

```bash
# 1. Start LocalStack + Shopping Cart API (all-in-one script)
cd ../shopping-cart-api
./run-local.sh
# This will:
# - Start LocalStack if not running
# - Verify DynamoDB is available
# - Start the shopping cart API on port 8080
# - Show logs in the terminal

# In a NEW terminal window, verify services are running:
# Terminal 2:
curl -s http://localhost:4566/_localstack/health | grep dynamodb
# Should show: "dynamodb":"running"

curl http://localhost:8080/health
# Should return: {"service":"shopping-cart-api","status":"healthy"}

# 2. Run baseline test (in Terminal 2)
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/dynamodb/tests
export BASE_URL=http://localhost:8080
go test -v -run TestDynamoDBPerformance -timeout 10m

# Output: dynamodb_test_results.json
# Rename for clarity:
mv dynamodb_test_results.json localstack_baseline_results.json

# 3. When done, stop the API (go back to Terminal 1 and press Ctrl+C)
```

### Step 1B: Test AWS

```bash
# 1. Ensure AWS infrastructure is deployed
cd ../terraform
terraform apply

# 2. Test automatically gets ALB URL from Terraform
cd ../tests

# 3. Run baseline test (no BASE_URL needed for AWS)
go test -v -run TestDynamoDBPerformance -timeout 10m

# Output: dynamodb_test_results.json
# Rename for clarity:
mv dynamodb_test_results.json aws_baseline_results.json
```

**Expected Output:**

```
=== Performance Test Summary ===

CREATE_CART:
  Total: 50
  Successful: 50 (100.00%)
  Response Time:
    Average: 45.23 ms
    Min: 12.45 ms
    Max: 234.67 ms

ADD_ITEMS:
  ...

OVERALL:
  Total Operations: 150
  Successful: 150 (100.00%)
  Average Response Time: 52.14 ms
```

---

## 🔥 Experiment 3: Flash Sale Load Test (High Concurrency)

This test simulates a realistic flash sale scenario with 500 concurrent users hammering your API for 1 minute using Locust.

### Prerequisites

Install Locust (if not already installed):

```bash
pip install locust
```

### Step 3A: Test LocalStack

```bash
# 1. Ensure LocalStack + API are running (from Experiment 1)
# If not, start them:
cd ../shopping-cart-api
./run-local.sh
# (Run this in a separate terminal)

# 2. In your test terminal:
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/dynamodb/tests

# 3. Run flash sale test (headless mode)
export BASE_URL=http://localhost:8080
python flash_sale_locust.py

# Output: flash_sale_localstack_results.json
```

### Step 3B: Test AWS

```bash
# 1. Ensure AWS infrastructure is deployed
cd ../terraform
terraform apply

# 2. Get the ALB URL and run flash sale test
cd ../tests
export BASE_URL=http://$(cd ../terraform && terraform output -raw alb_url)
python flash_sale_locust.py

# Output: flash_sale_aws_results.json
```

**Expected Output:**

```
======================================================================
               FLASH SALE LOAD TEST - 500 USERS
======================================================================

Environment: localstack
Base URL: http://localhost:8080
Duration: 60 seconds
Concurrent Users: 500
Spawn Rate: 200 users/second
Output Directory: flash_sale_results_localstack_20251128_143022

Starting test...

✓ Test completed in 60.12 seconds

Generating reports...

======================================================================
                    FLASH SALE TEST SUMMARY
======================================================================

Environment: localstack
Duration: 60 seconds
Total Users: 500
Total Operations: 6545

--------------------------------------------------
SUCCESS METRICS
--------------------------------------------------
Success Rate:  99.45%
Successful:    6509 ops
Failed:        36 ops

--------------------------------------------------
THROUGHPUT METRICS
--------------------------------------------------
Throughput:    108.89 ops/s
Avg Latency:   45.67 ms

--------------------------------------------------
LATENCY PERCENTILES
--------------------------------------------------
Min:           12.00 ms
P50 (Median):  42.00 ms
P90:           78.00 ms
P95:           95.00 ms
P99:           145.00 ms
Max:           234.00 ms

----------------------------------------------------------------------
OPERATION BREAKDOWN
----------------------------------------------------------------------
Operation       Count  Success   Avg ms   P95 ms   P99 ms
----------------------------------------------------------------------
CREATE_CART      2100    99.52%    38.45    82.34   125.67
ADD_ITEM         3145    99.12%    46.78    98.90   152.34
GET_CART         1300    99.85%    42.34    89.45   138.90
======================================================================

======================================================================
                         RESULTS SAVED
======================================================================

📁 Output Directory: flash_sale_results_localstack_20251128_143022
📊 HTML Report: flash_sale_results_localstack_20251128_143022/report.html
📋 JSON Results: flash_sale_results_localstack_20251128_143022/flash_sale_localstack_results.json
📈 CSV Stats: flash_sale_results_localstack_20251128_143022/stats.csv
📉 CSV History: flash_sale_results_localstack_20251128_143022/stats_history.csv
❌ Failures: flash_sale_results_localstack_20251128_143022/failures.csv
📝 Test Log: flash_sale_results_localstack_20251128_143022/test.log

======================================================================
```

**Key Features:**

- ✅ **Locust-powered**: Industry-standard load testing tool
- ✅ **HTML report with charts**: Beautiful visualization of results
- ✅ **Timestamped folder**: All results organized in one directory
- ✅ **Multiple formats**: JSON, CSV, HTML, and logs
- ✅ **Detailed statistics**: Success rate, latency percentiles, per-operation breakdown
- ✅ **Headless**: Runs without any UI, perfect for CI/CD
- ✅ **Interactive charts**: Request distribution, response times, percentiles
- ✅ **500 concurrent users**: Realistic high-load scenario
- ✅ **1 minute duration**: Quick but meaningful test

**Output Files:**

- `report.html` - Interactive HTML report with charts (open in browser)
- `flash_sale_{environment}_results.json` - Structured test results
- `stats.csv` - Detailed statistics for each operation
- `stats_history.csv` - Time-series data for performance over time
- `failures.csv` - Failed requests (if any)
- `test.log` - Complete test execution log

---

## 📊 Experiment 4: Cold Start Comparison

### Step 4A: Test LocalStack

```bash
# 1. Stop everything completely
cd ../shopping-cart-api
# If run-local.sh is running in another terminal, press Ctrl+C there first
docker-compose down
# Or: docker stop localstack && docker rm localstack

# 2. Start timer, then start everything fresh
# Terminal 1:
time ./run-local.sh
# This will start LocalStack AND the API
# Note: LocalStack startup takes ~10 seconds, API starts immediately after

# 3. In Terminal 2, once you see "Starting Shopping Cart API" in Terminal 1:
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/dynamodb/tests
export BASE_URL=http://localhost:8080

# Wait until health check succeeds:
curl http://localhost:8080/health

# 4. Run cold start test
go test -v -run TestColdStart -timeout 5m

# Output: cold_start_localstack_results.json
```

### Step 4B: Test AWS

**⚠️ WARNING: This will tear down your infrastructure**

```bash
# 1. Destroy infrastructure
cd ../terraform
terraform destroy -auto-approve

# 2. Start a timer, then deploy
time terraform apply -auto-approve

# 3. Run cold start test immediately (while timer running)
cd ../tests
go test -v -run TestColdStart -timeout 5m

# Output: cold_start_aws_results.json
```

**Expected Output:**

````
╔════════════════════════════════════════╗
║  Testing Load Level: 150 operations    ║
╚════════════════════════════════════════╝

--- Results for 150 operations ---
Duration: 12.45 seconds
Throughput: 12.05 ops/sec
Success Rate: 100.00%
P95: 89.23 ms

Waiting 10 seconds before next load level...

╔════════════════════════════════════════╗
║  Testing Load Level: 300 operations    ║
╚════════════════════════════════════════╝
### Step 3A: Test LocalStack

```bash
# 1. Stop everything completely
cd ../shopping-cart-api
# If run-local.sh is running in another terminal, press Ctrl+C there first
docker-compose down
# Or: docker stop localstack && docker rm localstack

# 2. Start timer, then start everything fresh
# Terminal 1:
time ./run-local.sh
# This will start LocalStack AND the API
# Note: LocalStack startup takes ~10 seconds, API starts immediately after

# 3. In Terminal 2, once you see "Starting Shopping Cart API" in Terminal 1:
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/dynamodb/tests
export BASE_URL=http://localhost:8080

# Wait until health check succeeds:
curl http://localhost:8080/health

# 4. Run cold start test
go test -v -run TestColdStart -timeout 5m

# Output: cold_start_localstack_results.json
````

# 4. Run cold start test immediately after API is ready

export BASE_URL=http://localhost:8080
cd ../tests
go test -v -run TestColdStart -timeout 5m

# Output: cold_start_localstack_results.json

````

### Step 3B: Test AWS

**⚠️ WARNING: This will tear down your infrastructure**

```bash
# 1. Destroy infrastructure
cd ../terraform
terraform destroy -auto-approve

# 2. Start a timer, then deploy
time terraform apply -auto-approve

# 3. Run cold start test immediately (while timer running)
cd ../tests
go test -v -run TestColdStart -timeout 5m

# Output: cold_start_aws_results.json
````

**Expected Output:**

```
╔════════════════════════════════════════╗
║      COLD START MEASUREMENT TEST      ║
╚════════════════════════════════════════╝

Waiting for service to become ready...
Attempt 1/60... Status: 503
Attempt 2/60... Status: 503
Attempt 3/60... ✓ Success after 15.23 seconds

Attempting first create cart request...
✓ First request succeeded (45.67 ms)

Verifying stability with 5 consecutive requests...
  Request 1: ✓
  Request 2: ✓
  ...

╔════════════════════════════════════════════════════╗
║            COLD START SUMMARY                      ║
╚════════════════════════════════════════════════════╝

Environment: aws
Infrastructure Up Time: 15.23 seconds
Time to First Successful Request: 0.05 seconds
Total Cold Start Time: 15.23 seconds
```

## 🔍 Experiment 5: Consistency Testing

```bash
# Ensure LocalStack + API are running
# If not: cd ../shopping-cart-api && ./run-local.sh (in separate terminal)

# In test terminal:
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/dynamodb/tests
export BASE_URL=http://localhost:8080

# Run consistency tests
go test -v -run TestCreateThenReadConsistency -timeout 10m
go test -v -run TestAddItemThenReadConsistency -timeout 10m
go test -v -run TestRapidConcurrentUpdates -timeout 10m

# Save output for LocalStack
# (Tests write to stdout, redirect if needed)

# For AWS (auto-detects URL):
# Ensure AWS infrastructure is deployed first
go test -v -run TestCreateThenReadConsistency -timeout 10m
go test -v -run TestAddItemThenReadConsistency -timeout 10m
go test -v -run TestRapidConcurrentUpdates -timeout 10m
```

---

## ❌ Experiment 6: Failure Mode Testing

### Step 6A: Test LocalStack

```bash
# Ensure LocalStack + API are running
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/dynamodb/tests
export BASE_URL=http://localhost:8080

# Run all failure tests
go test -v -run TestInvalidRequests -timeout 5m
go test -v -run TestMissingResources -timeout 5m
go test -v -run TestMalformedData -timeout 5m

# Rename outputs:
mv failure_mode_invalid_requests_results.json localstack_failure_invalid_requests.json
mv failure_mode_missing_resources_results.json localstack_failure_missing_resources.json
mv failure_mode_malformed_data_results.json localstack_failure_malformed_data.json
```

### Step 6B: Test AWS

mv failure_mode_invalid_requests_results.json localstack_failure_invalid_requests.json
mv failure_mode_missing_resources_results.json localstack_failure_missing_resources.json
mv failure_mode_malformed_data_results.json localstack_failure_malformed_data.json

`````

### Step 6B: Test AWS

```bash
# Run all failure tests (auto-detects AWS URL)
go test -v -run TestInvalidRequests -timeout 5m
go test -v -run TestMissingResources -timeout 5m
go test -v -run TestMalformedData -timeout 5m

# Rename outputs:
mv failure_mode_invalid_requests_results.json aws_failure_invalid_requests.json
mv failure_mode_missing_resources_results.json aws_failure_missing_resources.json
mv failure_mode_malformed_data_results.json aws_failure_malformed_data.json
```
go test -v -run TestMissingResources -timeout 5m
go test -v -run TestMalformedData -timeout 5m

# Rename outputs:
mv failure_mode_invalid_requests_results.json aws_failure_invalid_requests.json
mv failure_mode_missing_resources_results.json aws_failure_missing_resources.json
mv failure_mode_malformed_data_results.json aws_failure_malformed_data.json
```

---

## 📊 Results Analysis

After running all tests, you'll have these files:

### Baseline Results

- `localstack_baseline_results.json`
- `aws_baseline_results.json`

### Load Scaling Results

- `localstack_load_scaling_results.json`
- `aws_load_scaling_results.json`

### Flash Sale Results

- `flash_sale_localstack_results.json`
- `flash_sale_aws_results.json`

### Cold Start Results

- `cold_start_localstack_results.json`
- `cold_start_aws_results.json`

### Consistency Results

- `localstack_consistency_results.txt`
- `aws_consistency_results.txt`

### Failure Mode Results

- `localstack_failure_*.json` (3 files)
- `aws_failure_*.json` (3 files)

---

## 📈 Analysis Script

Create a Python script to analyze all results:

```python
# analysis/compare_results.py
import json
import glob

# Load all JSON results
localstack_files = glob.glob("localstack_*.json")
aws_files = glob.glob("aws_*.json")

# Compare metrics...
# (Add your analysis code)
```

---

## 🎯 Key Metrics Summary Table

After testing, create this comparison:

| Metric                   | LocalStack   | AWS          | Difference |
| ------------------------ | ------------ | ------------ | ---------- |
| **Baseline Avg Latency** | \_\_\_ ms    | \_\_\_ ms    | \_\_\_     |
| **Baseline P95**         | \_\_\_ ms    | \_\_\_ ms    | \_\_\_     |
| **Baseline P99**         | \_\_\_ ms    | \_\_\_ ms    | \_\_\_     |
| **Throughput (150 ops)** | \_\_\_ ops/s | \_\_\_ ops/s | \_\_\_     |
| **Throughput (500 ops)** | \_\_\_ ops/s | \_\_\_ ops/s | \_\_\_     |

### Test hangs or fails to connect

````bash
# For LocalStack:
# 1. Check if run-local.sh is running in another terminal
#    You should see log output from the Go application

# 2. Check LocalStack DynamoDB is running:
curl -s http://localhost:4566/_localstack/health | grep dynamodb
# Should show: "dynamodb":"running"

# 3. Check shopping cart API is running:
curl http://localhost:8080/health
# Should return: {"service":"shopping-cart-api","status":"healthy"}

# 4. If API not responding:
#    - Check if port 8080 is in use: lsof -i :8080
#    - Check LocalStack logs: docker logs localstack
#    - Restart: Ctrl+C in terminal running ./run-local.sh, then restart it

# For AWS:
curl $(cd ../terraform && terraform output -raw alb_url)/health
```l http://localhost:8080/health
# Should return: {"status":"healthy"}

# 3. Check LocalStack container logs:
cd ../shopping-cart-api
docker-compose logs -f

# For AWS:
curl $(cd ../terraform && terraform output -raw alb_url)/health
`````

### "Failed to get terraform output"

```bash
# Manually set URL
export BASE_URL=http://your-alb-url.amazonaws.com

# Or check Terraform
cd ../terraform
terraform output alb_url
```

### Tests timeout

```bash
# Increase timeout
go test -v -run TestName -timeout 20m
```

---

## ✅ Checklist

Before you finish, ensure you have:

- [ ] `localstack_baseline_results.json`
- [ ] `aws_baseline_results.json`
- [ ] `localstack_load_scaling_results.json`
- [ ] `aws_load_scaling_results.json`
- [ ] `flash_sale_localstack_results.json`
- [ ] `flash_sale_aws_results.json`
- [ ] `cold_start_localstack_results.json`
- [ ] `cold_start_aws_results.json`
- [ ] `localstack_consistency_results.txt`
- [ ] `aws_consistency_results.txt`
- [ ] 6 failure mode JSON files (3 LocalStack + 3 AWS)

**Total: 16 result files** 📁

---

## 🎓 Report Writing Tips

### Introduction

- Explain LocalStack vs AWS comparison goal
- Describe the shopping cart API architecture
- List the 5 experiments

### Methodology

- Describe each test suite
- Explain why Go tests were sufficient (no Locust needed)
- Detail the test environment (LocalStack Docker, AWS ECS + ALB)

### Results

- Present comparison tables for each experiment
- Include graphs showing:
  - Latency distribution (baseline)
  - Throughput vs load level (scaling)
  - Cold start times (bar chart)
  - Consistency patterns (timeline)
  - Error handling rates

### Discussion

- **Network latency impact** - LocalStack ~0ms vs AWS real network
- **Cold start differences** - Docker vs ECS task startup
- **Scaling behavior** - How each environment handles increased load
- **Consistency** - Both use DynamoDB, consistency should be similar
- **Error handling** - Should be identical (same code)

### Conclusion

- When to use LocalStack (development, CI/CD)
- When to use AWS (production)
- Cost-benefit analysis

---

## 📞 Need Help?

If tests fail or results are unexpected:

1. Check service health
2. Review test logs
3. Verify terraform state
4. Check docker logs (LocalStack)
5. Review CloudWatch logs (AWS)

Good luck with your testing! 🚀
