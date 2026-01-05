# Flash Sale Load Test

A Locust-based load test simulating a flash sale scenario with 500 concurrent users for 1 minute. Generates beautiful HTML reports with interactive charts.

## Features

- ✅ **500 concurrent users** - Realistic high-load scenario
- ✅ **1 minute duration** - Quick but meaningful test
- ✅ **Headless mode** - No UI required, perfect for CI/CD
- ✅ **HTML reports with charts** - Beautiful visualizations using Chart.js
- ✅ **Multiple output formats** - JSON, CSV, HTML, and logs
- ✅ **Timestamped folders** - Organized results for each test run
- ✅ **Works with both LocalStack and AWS** - Automatically detects environment

## Prerequisites

Install Locust:

```bash
pip install locust
```

## Usage

### Testing LocalStack

```bash
# Start LocalStack and API
cd ../shopping-cart-api
./run-local.sh

# In another terminal, run the test
cd ../tests
export BASE_URL=http://localhost:8080
python flash_sale_locust.py
```

### Testing AWS

```bash
# Deploy to AWS
cd ../terraform
terraform apply

# Run the test
cd ../tests
export BASE_URL=http://$(cd ../terraform && terraform output -raw alb_url)
python flash_sale_locust.py
```

## Configuration

Edit these constants in `flash_sale_locust.py` to adjust the test:

```python
NUM_USERS = 500          # Number of concurrent users
DURATION_SECONDS = 60    # Test duration in seconds
SPAWN_RATE = 200         # Users spawned per second
```

## User Behavior

Each simulated user:

1. Creates a shopping cart
2. Adds 1-3 random items (products 1000-1099)
3. Retrieves cart information
4. Waits 50-200ms between operations
5. Repeats until test duration expires

## Output

All results are saved to a timestamped directory: `flash_sale_results_{environment}_{timestamp}/`

### Directory Contents

```
flash_sale_results_localstack_20251128_143022/
├── report.html                           # Interactive HTML report with charts
├── flash_sale_localstack_results.json   # Structured test results
├── stats.csv                            # Detailed operation statistics
├── stats_history.csv                    # Time-series performance data
├── failures.csv                         # Failed requests (if any)
└── test.log                            # Complete test execution log
```

### HTML Report

Open `report.html` in any browser to see:

- **Summary metrics** - Users, requests, success rate, throughput, response times
- **Interactive charts**:
  - Request distribution across operations
  - Average response times by operation
  - P95 response times by operation
- **Detailed table** - Per-operation statistics with all percentiles

### Console Output

Minimal console output showing:

- Test configuration
- Progress notification
- Final summary with key metrics
- List of all generated files

### JSON Results Structure

The `flash_sale_{environment}_results.json` file includes:

- Summary statistics (throughput, success rate, latency)
- Per-operation breakdown (CREATE_CART, ADD_ITEM, GET_CART)
- Latency percentiles (P50, P90, P95, P99)
- Test configuration

### CSV Files

- `stats.csv` - Comprehensive statistics for each operation type
- `stats_history.csv` - Performance metrics over time (useful for graphing trends)
- `failures.csv` - Details of any failed requests

## Example JSON Results

```json
{
  "test_name": "Flash Sale Load Test",
  "environment": "localstack",
  "configuration": {
    "total_users": 500,
    "duration_seconds": 60,
    "spawn_rate": 50
  },
  "summary": {
    "total_operations": 6545,
    "success_count": 6509,
    "failure_count": 36,
    "success_rate_percent": 99.45,
    "total_throughput_ops_per_sec": 108.89,
    "avg_response_time_ms": 45.67
  },
  "latency_percentiles": {
    "min_ms": 12,
    "p50_ms": 42,
    "p90_ms": 78,
    "p95_ms": 95,
    "p99_ms": 145,
    "max_ms": 234
  },
  "operation_breakdown": {
    "CREATE_CART": {
      "count": 2100,
      "success_count": 2089,
      "avg_response_time_ms": 38.45,
      "p95_ms": 82.34,
      "p99_ms": 125.67
    },
    ...
  }
}
```

## Troubleshooting

### Service not responding

Check if the service is healthy:

```bash
curl $BASE_URL/health
```

### LocalStack issues

Check if LocalStack and API are running:

```bash
# Check LocalStack
curl -s http://localhost:4566/_localstack/health | grep dynamodb

# Check API
curl http://localhost:8080/health
```

### AWS issues

Verify terraform deployment:

```bash
cd ../terraform
terraform output alb_url
```

## Comparing Results

Compare LocalStack vs AWS performance using the JSON results from each test run folder:

```python
import json
from pathlib import Path

# Find the most recent result directories
localstack_dir = sorted(Path('.').glob('flash_sale_results_localstack_*'))[-1]
aws_dir = sorted(Path('.').glob('flash_sale_results_aws_*'))[-1]

# Load results
with open(localstack_dir / 'flash_sale_localstack_results.json') as f:
    localstack = json.load(f)

with open(aws_dir / 'flash_sale_aws_results.json') as f:
    aws = json.load(f)

# Compare throughput
print(f"LocalStack: {localstack['summary']['total_throughput_ops_per_sec']:.2f} ops/s")
print(f"AWS:        {aws['summary']['total_throughput_ops_per_sec']:.2f} ops/s")

# Compare latency
print(f"\nP95 Latency:")
print(f"LocalStack: {localstack['latency_percentiles']['p95_ms']:.2f} ms")
print(f"AWS:        {aws['latency_percentiles']['p95_ms']:.2f} ms")
```

## Viewing HTML Reports

Simply open the `report.html` file in any web browser:

```bash
# macOS
open flash_sale_results_localstack_*/report.html

# Linux
xdg-open flash_sale_results_localstack_*/report.html

# Windows
start flash_sale_results_localstack_*/report.html
```

The HTML report includes interactive charts and detailed tables for easy analysis.
