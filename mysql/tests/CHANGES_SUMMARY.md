# MySQL Tests - Summary of Changes

## ✅ What Was Added

Your MySQL tests directory has been enhanced with **4 new test files** to match the comprehensive testing structure from your DynamoDB implementation, while respecting your existing baseline test.

### New Test Files Created

1. **`load_scaling_test.go`** ⭐ NEW

   - Progressive load testing (150 → 300 → 500 operations)
   - Measures throughput degradation under increasing load
   - Calculates P50, P95, P99 latencies at each load level
   - Outputs: `localstack_load_scaling_results.json` or `aws_load_scaling_results.json`

2. **`cold_start_test.go`** ⭐ NEW

   - Measures time from infrastructure startup to first successful request
   - Tests service stability with consecutive requests
   - Tracks health check latency
   - Outputs: `cold_start_localstack_results.json` or `cold_start_aws_results.json`

3. **`consistency_test.go`** ⭐ NEW

   - Tests read-after-write consistency patterns
   - Includes 4 test scenarios:
     - Create-then-read consistency
     - Add-item-then-read consistency
     - Rapid concurrent updates
     - Write-read-write patterns
   - Outputs: `{env}_consistency_{test}_results.json`

4. **`failure_mode_test.go`** ⭐ NEW

   - Validates error handling for invalid requests
   - Tests 3 categories:
     - Invalid requests (missing fields, wrong types, negative values)
     - Missing resources (404 errors)
     - Malformed data (extra fields, large values, empty JSON)
   - Outputs: `{env}_failure_{category}_results.json`

5. **`COMPREHENSIVE_TESTING_GUIDE.md`** ⭐ NEW
   - Complete documentation for running all tests
   - Step-by-step instructions for LocalStack and AWS
   - Troubleshooting guide
   - Comparison checklist

---

## ✅ What Was Kept

- **`performance.go`** - Your existing baseline test (150 operations)
- **`mysql_test_results.json`** - Your existing test results
- **`result.txt`** - Your existing test output

---

## 🎯 Key Features

### Environment Detection

All new tests automatically detect whether they're running against:

- **LocalStack**: `http://localhost:3000` (from `API_URL` environment variable)
- **AWS**: ALB URL from Terraform output (`terraform output -raw alb_url`)

### URL Configuration Priority

1. Try to get ALB URL from Terraform (for AWS)
2. Fall back to `API_URL` environment variable
3. Fall back to `http://localhost:3000` (LocalStack default)

### Test Structure

All tests follow the same pattern as your DynamoDB tests:

- Go test files (can run with `go test -v -run TestName`)
- JSON output files with detailed metrics
- Automatic environment detection
- Error handling and retry logic
- Progress indicators and summaries

---

## 🚀 Quick Start Commands

### LocalStack Testing

```bash
# Terminal 1: Start services
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/mysql
./run-local.sh

# Terminal 2: Run tests
cd tests
export API_URL=http://localhost:3000

# Baseline test (existing)
go run performance.go

# New tests
go test -v -run TestLoadScaling -timeout 20m
go test -v -run TestColdStart -timeout 10m
go test -v -run TestCreateThenReadConsistency -timeout 5m
go test -v -run TestInvalidRequests -timeout 5m
```

### AWS Testing

```bash
# Deploy infrastructure
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/final/mysql/terraform
terraform apply

# Run tests (automatically gets ALB URL)
cd ../tests

# Baseline test (existing)
go run performance.go

# New tests
go test -v -run TestLoadScaling -timeout 20m
go test -v -run TestColdStart -timeout 10m
go test -v -run TestCreateThenReadConsistency -timeout 5m
go test -v -run TestInvalidRequests -timeout 5m
```

---

## 📊 Expected Output Files

After running all tests, you'll have:

### LocalStack

```
localstack_baseline_results.json                          # Baseline 150-op test
localstack_load_scaling_results.json                      # Load scaling test
cold_start_localstack_results.json                        # Cold start test
localstack_consistency_create_then_read_results.json      # Consistency tests
localstack_consistency_add_item_then_read_results.json
localstack_failure_invalid_requests_results.json          # Failure mode tests
localstack_failure_missing_resources_results.json
localstack_failure_malformed_data_results.json
```

### AWS

```
aws_baseline_results.json                                 # Baseline 150-op test
aws_load_scaling_results.json                             # Load scaling test
cold_start_aws_results.json                               # Cold start test
aws_consistency_create_then_read_results.json             # Consistency tests
aws_consistency_add_item_then_read_results.json
aws_failure_invalid_requests_results.json                 # Failure mode tests
aws_failure_missing_resources_results.json
aws_failure_malformed_data_results.json
```

---

## 🔍 Differences from DynamoDB Tests

1. **No Flash Sale Test** ❌ - Excluded as you requested
2. **Port 3000** - MySQL API runs on port 3000 (DynamoDB uses 8080)
3. **API_URL variable** - Uses `API_URL` instead of `BASE_URL`
4. **Different data types** - Cart IDs are `int` not `int64`
5. **Strong consistency** - MySQL has strong consistency (vs DynamoDB eventual consistency)

---

## 📁 File Structure

```
mysql/tests/
├── performance.go                        # ✅ Existing baseline test
├── load_scaling_test.go                  # ⭐ NEW
├── cold_start_test.go                    # ⭐ NEW
├── consistency_test.go                   # ⭐ NEW
├── failure_mode_test.go                  # ⭐ NEW
├── COMPREHENSIVE_TESTING_GUIDE.md        # ⭐ NEW
├── mysql_test_results.json               # ✅ Existing results
└── result.txt                            # ✅ Existing output
```

---

## ✅ Ready to Test!

Your MySQL tests are now complete and ready to run against both LocalStack and AWS environments. Follow the `COMPREHENSIVE_TESTING_GUIDE.md` for detailed instructions.

All tests will work seamlessly with your LocalStack setup:

- ✅ LocalStack Pro (v4.11.2.dev5) - Port 4566
- ✅ RDS MySQL (8.0.40) - Instance: shopping-cart-mysql
- ✅ MySQL Container - ls-mysql-046d0024 with database shopping_cart_db
- ✅ Shopping Cart API - Port 3000, Status: Healthy

Happy testing! 🚀
