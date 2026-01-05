#!/bin/bash
# MySQL Tests Runner - Run all tests in sequence
# Usage: ./run_all_tests.sh [localstack|aws]

set -e  # Exit on error

ENVIRONMENT=${1:-localstack}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "════════════════════════════════════════════════════════════"
echo "  MySQL Comprehensive Test Suite"
echo "  Environment: $ENVIRONMENT"
echo "════════════════════════════════════════════════════════════"
echo ""

# Set API URL based on environment
if [ "$ENVIRONMENT" = "localstack" ]; then
    export API_URL="http://localhost:3000"
    echo "✓ Using LocalStack: $API_URL"
    
    # Check if LocalStack services are running
    echo "⏳ Checking LocalStack services..."
    if ! curl -s http://localhost:3000/health > /dev/null 2>&1; then
        echo "❌ API not reachable at http://localhost:3000"
        echo ""
        echo "Please start LocalStack services first:"
        echo "  cd .. && ./run-local.sh"
        exit 1
    fi
    echo "✓ API is healthy"
    echo ""
    
elif [ "$ENVIRONMENT" = "aws" ]; then
    echo "⏳ Getting ALB URL from Terraform..."
    cd "$SCRIPT_DIR/../terraform"
    
    if ! terraform output > /dev/null 2>&1; then
        echo "❌ Terraform outputs not available"
        echo ""
        echo "Please deploy AWS infrastructure first:"
        echo "  cd ../terraform && terraform apply"
        exit 1
    fi
    
    ALB_URL=$(terraform output -raw alb_url 2>/dev/null)
    if [ -z "$ALB_URL" ]; then
        echo "❌ Could not get ALB URL from Terraform"
        exit 1
    fi
    
    export API_URL="http://$ALB_URL"
    echo "✓ Using AWS: $API_URL"
    
    cd "$SCRIPT_DIR"
    
    # Check if API is reachable
    echo "⏳ Checking API connectivity..."
    if ! curl -s "$API_URL/health" > /dev/null 2>&1; then
        echo "❌ API not reachable at $API_URL"
        echo "Please check AWS infrastructure"
        exit 1
    fi
    echo "✓ API is healthy"
    echo ""
else
    echo "❌ Invalid environment: $ENVIRONMENT"
    echo "Usage: $0 [localstack|aws]"
    exit 1
fi

# Ensure we're in the tests directory
cd "$SCRIPT_DIR"

# Initialize Go module if needed
if [ ! -f "go.mod" ]; then
    echo "⏳ Initializing Go module..."
    go mod init mysql-tests
fi

echo "════════════════════════════════════════════════════════════"
echo "  Test 1: Baseline Performance Test (150 operations)"
echo "════════════════════════════════════════════════════════════"
echo ""
go run performance.go
if [ -f "mysql_test_results.json" ]; then
    mv mysql_test_results.json "${ENVIRONMENT}_baseline_results.json"
    echo "✓ Saved to: ${ENVIRONMENT}_baseline_results.json"
fi
echo ""

echo "════════════════════════════════════════════════════════════"
echo "  Test 2: Load Scaling Test"
echo "════════════════════════════════════════════════════════════"
echo ""
go test -v -run TestLoadScaling -timeout 20m
echo ""

echo "════════════════════════════════════════════════════════════"
echo "  Test 3: Consistency Tests"
echo "════════════════════════════════════════════════════════════"
echo ""
echo "→ Test 3.1: Create-then-Read Consistency"
go test -v -run TestCreateThenReadConsistency -timeout 5m
echo ""

echo "→ Test 3.2: Add-Item-then-Read Consistency"
go test -v -run TestAddItemThenReadConsistency -timeout 5m
echo ""

echo "→ Test 3.3: Rapid Concurrent Updates"
go test -v -run TestRapidConcurrentUpdates -timeout 5m
echo ""

echo "→ Test 3.4: Write-Read-Write Pattern"
go test -v -run TestWriteReadWritePattern -timeout 5m
echo ""

echo "════════════════════════════════════════════════════════════"
echo "  Test 4: Failure Mode Tests"
echo "════════════════════════════════════════════════════════════"
echo ""
echo "→ Test 4.1: Invalid Requests"
go test -v -run TestInvalidRequests -timeout 5m
echo ""

echo "→ Test 4.2: Missing Resources"
go test -v -run TestMissingResources -timeout 5m
echo ""

echo "→ Test 4.3: Malformed Data"
go test -v -run TestMalformedData -timeout 5m
echo ""

# Show summary of generated files
echo ""
echo "════════════════════════════════════════════════════════════"
echo "  ✅ All Tests Complete!"
echo "════════════════════════════════════════════════════════════"
echo ""
echo "Generated test result files:"
echo "───────────────────────────────────────────────────────────"
ls -lh "${ENVIRONMENT}"*.json 2>/dev/null | awk '{print "  " $9 " (" $5 ")"}'
echo "───────────────────────────────────────────────────────────"
echo ""
echo "Next steps:"
echo "  1. Review the JSON result files"
echo "  2. Run tests for the other environment (localstack/aws)"
echo "  3. Compare results between environments"
echo "  4. Compare with DynamoDB results"
echo ""

if [ "$ENVIRONMENT" = "localstack" ]; then
    echo "To test AWS environment:"
    echo "  $0 aws"
else
    echo "To test LocalStack environment:"
    echo "  $0 localstack"
fi
echo ""
