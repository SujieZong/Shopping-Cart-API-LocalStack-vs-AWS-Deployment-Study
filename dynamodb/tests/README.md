# DynamoDB Consistency Tests

This directory contains tests to evaluate eventual consistency behavior of the DynamoDB-based shopping cart API.

## Automatic Base URL Configuration

The tests now automatically retrieve the ALB URL from Terraform output, so you don't need to manually update the `baseURL` variable anymore!

### How it works:

1. The test automatically runs `terraform output -raw alb_url` from the `../terraform` directory
2. It uses the retrieved URL as the base URL for all API calls
3. If Terraform output fails, it falls back to the `BASE_URL` environment variable

### Running the Tests

**Option 1: Automatic (Recommended)**

```bash
cd /Users/sujiezong/Desktop/NEU/6650_distributed_system/6650_assignments/CS6650-HW/HW8/dynamodb/tests

# Run as Go tests
go test -v

# Or run as standalone program
go run consistency_test.go
```

**Option 2: With Environment Variable (Fallback)**

```bash
export BASE_URL=http://your-alb-url.amazonaws.com
go test -v
```

### Prerequisites

1. Terraform infrastructure must be deployed
2. Run tests from the `tests` directory (or ensure relative path `../terraform` exists)
3. Terraform state must contain the `alb_url` output

### Test Scenarios

The test suite includes:

1. **Create-then-Read Consistency**: Tests immediate read after cart creation
2. **Add-Item-then-Read Consistency**: Tests reading items immediately after adding them
3. **Rapid Concurrent Updates**: Tests behavior under concurrent write operations
4. **Write-Read-Write Pattern**: Tests read-modify-write scenarios

### Troubleshooting

**Error: "Failed to get base URL from Terraform"**

- Ensure you're running from the `tests` directory
- Verify Terraform is installed: `terraform version`
- Check that Terraform state exists: `ls ../terraform/terraform.tfstate`
- Manually set BASE_URL: `export BASE_URL=http://your-alb-url.amazonaws.com`

**Error: "terraform output returned empty URL"**

- Run `terraform output alb_url` manually from the terraform directory
- Ensure Terraform infrastructure is deployed: `cd ../terraform && terraform apply`

### Manual URL Retrieval

If you need to see the URL manually:

```bash
cd ../terraform
terraform output alb_url
```
