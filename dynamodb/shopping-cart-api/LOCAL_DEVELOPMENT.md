# Local Development Guide

## 🚀 Quick Start

**One command to start everything:**

```bash
./run-local.sh
```

This script will:

1. ✅ Start LocalStack (if not already running)
2. ✅ Verify DynamoDB is available
3. ✅ Start the Shopping Cart API on port 8080
4. ✅ Show logs in real-time

Press `Ctrl+C` to stop the API.

---

## ✅ Verify Everything is Running

**In a separate terminal:**

```bash
# Check LocalStack DynamoDB
curl -s http://localhost:4566/_localstack/health | grep dynamodb
# Should show: "dynamodb":"running"

# Check Shopping Cart API
curl http://localhost:8080/health
# Should return: {"service":"shopping-cart-api","status":"healthy"}
```

---

## 🧪 Run Tests

Once the API is running, open a new terminal:

```bash
cd ../tests
export BASE_URL=http://localhost:8080

# Run baseline test
go test -v -run TestDynamoDBPerformance -timeout 10m

# Run all tests
go test -v -timeout 30m
```

---

## 🛠️ Manual Setup (Advanced)

If you want more control, you can start components separately:

### Option 1: Start LocalStack only

```bash
./setup-localstack.sh
```

### Option 2: Start API only (LocalStack must be running)

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-west-2
export DYNAMODB_TABLE_NAME=ShoppingCarts
export PORT=8080

go run *.go
```

---

## 🔧 Troubleshooting

### Port 8080 is already in use

```bash
# Find what's using port 8080
lsof -i :8080

# Kill the process
lsof -ti:8080 | xargs kill -9

# Or the script will ask you automatically
./run-local.sh
```

### LocalStack not starting

```bash
# Check Docker is running
docker info

# Restart LocalStack
docker restart localstack

# Or recreate it
docker stop localstack && docker rm localstack
./setup-localstack.sh
```

### API won't connect to LocalStack

```bash
# Check LocalStack health
curl http://localhost:4566/_localstack/health

# Check LocalStack logs
docker logs localstack

# Restart everything
docker restart localstack
./run-local.sh
```

---

## 📁 Files Explained

| File                            | Purpose                             |
| ------------------------------- | ----------------------------------- |
| `run-local.sh`                  | 🚀 Main script - starts everything  |
| `setup-localstack.sh`           | Sets up LocalStack infrastructure   |
| `setup-localstack-table.sh`     | Creates DynamoDB table              |
| `docker-compose.localstack.yml` | Docker Compose config (alternative) |

---

## 🎯 What Changed?

**Old behavior:** `run-local.sh` only checked LocalStack, didn't start the API  
**New behavior:** `run-local.sh` starts LocalStack AND the API ✅

Now you just need **one command** to start developing locally!
