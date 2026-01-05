# Quick Start Guide

## Build & Run Locally

1. **Install Go 1.21+**

2. **Set up AWS credentials:**

```bash
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
export AWS_REGION=us-west-2
```

3. **Build:**

```bash
go build
```

4. **Run:**

```bash
./shopping-cart-api
```

## Test API

```bash
# Create cart
curl -X POST http://localhost:8080/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 12345}'

# Add item (use cart ID from previous response)
curl -X POST http://localhost:8080/shopping-carts/1698765432567/items \
  -H "Content-Type: application/json" \
  -d '{"product_id": 100, "quantity": 2}'

# Get cart
curl http://localhost:8080/shopping-carts/1698765432567
```

## Docker

```bash
docker build -t shopping-cart-api .
docker run -p 8080:8080 shopping-cart-api
```

## Deploy to AWS

See `../terraform/main.tf` - infrastructure is already configured!

```bash
cd ../terraform
terraform apply
```
