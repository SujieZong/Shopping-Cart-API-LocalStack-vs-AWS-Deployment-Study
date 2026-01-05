# LocalStack Local Development Setup

This directory contains configuration for running the Shopping Cart API locally using **LocalStack Pro** with RDS MySQL support.

## Why LocalStack?

LocalStack simulates AWS services locally, providing:

- ✅ **Real AWS RDS experience** without AWS costs
- ✅ **Test infrastructure code** before deploying to AWS
- ✅ **Faster development** with local AWS services
- ✅ **Offline development** capability

## Prerequisites

1. **Docker Desktop** installed and running
2. **LocalStack Pro account** (you mentioned you're a Pro user)
3. **AWS CLI Local** (awslocal) - install with:
   ```bash
   pip install awscli-local
   ```

## Quick Start

### 1. Set Your LocalStack Auth Token

```bash
# Set the token for current session
export LOCALSTACK_AUTH_TOKEN=your-token-here

# Or add to your shell profile for persistence
echo 'export LOCALSTACK_AUTH_TOKEN=your-token-here' >> ~/.zshrc
source ~/.zshrc
```

Get your token from: https://app.localstack.cloud/account/apikeys

### 2. Run the Application

```bash
./run-local.sh
```

That's it! The script will:

1. Start LocalStack with RDS support
2. Create a MySQL RDS instance
3. Initialize the database schema
4. Start the API service
5. Run test requests

## How to Exit/Stop

```bash
# Stop all services
docker-compose -f docker-compose.localstack.yml down

# Stop and remove all data (clean restart)
docker-compose -f docker-compose.localstack.yml down -v
```

## Architecture

```
┌─────────────────────────────────────────┐
│         Your Application                │
│      (Shopping Cart API)                │
│         Port: 3000                      │
└──────────────┬──────────────────────────┘
               │
               │ Connects to
               │
┌──────────────▼──────────────────────────┐
│         LocalStack Pro                  │
│         Port: 4566                      │
│                                         │
│   ┌─────────────────────────────┐      │
│   │   RDS MySQL Instance        │      │
│   │   Port: 43306               │      │
│   │   DB: shopping_cart_db      │      │
│   └─────────────────────────────┘      │
└─────────────────────────────────────────┘
```

## Files

- `docker-compose.localstack.yml` - Docker Compose config with LocalStack + API
- `setup-localstack.sh` - Initializes LocalStack and creates RDS instance
- `setup-localstack-db.sh` - Database schema initialization (auto-runs)
- `run-local.sh` - One-command startup script

## Useful Commands

### Check Status

```bash
# LocalStack health
curl http://localhost:4566/_localstack/health

# RDS instance status
awslocal rds describe-db-instances --db-instance-identifier shopping-cart-mysql

# API health
curl http://localhost:3000/health
```

### Access MySQL

```bash
# Get RDS endpoint
RDS_ENDPOINT=$(awslocal rds describe-db-instances \
  --db-instance-identifier shopping-cart-mysql \
  --query 'DBInstances[0].Endpoint.Address' \
  --output text)

# Connect to MySQL
docker exec shopping-cart-localstack mysql \
  -h ${RDS_ENDPOINT} -P 43306 \
  -u admin -pMySecurePass123! shopping_cart_db
```

### View Logs

```bash
# API logs
docker logs shopping-cart-api -f

# LocalStack logs
docker logs shopping-cart-localstack -f

# All logs
docker-compose -f docker-compose.localstack.yml logs -f
```

### Restart Services

```bash
# Restart API only
docker-compose -f docker-compose.localstack.yml restart api

# Restart LocalStack only
docker-compose -f docker-compose.localstack.yml restart localstack

# Full clean restart
docker-compose -f docker-compose.localstack.yml down -v
./run-local.sh
```

## Troubleshooting

### LocalStack Auth Issues

```bash
# Verify token is set
echo $LOCALSTACK_AUTH_TOKEN

# Check LocalStack logs for auth errors
docker logs shopping-cart-localstack | grep -i auth
```

### RDS Not Available

```bash
# Check RDS status
awslocal rds describe-db-instances

# Recreate RDS instance
./setup-localstack.sh
```

### API Can't Connect to Database

```bash
# Check API logs
docker logs shopping-cart-api

# Verify RDS endpoint
awslocal rds describe-db-instances \
  --db-instance-identifier shopping-cart-mysql \
  --query 'DBInstances[0].Endpoint'

# Restart API with fresh connection
docker-compose -f docker-compose.localstack.yml restart api
```

### Port Already in Use

```bash
# Check what's using port 4566
lsof -i :4566

# Check what's using port 3000
lsof -i :3000

# Kill processes if needed
lsof -ti:4566 | xargs kill -9
lsof -ti:3000 | xargs kill -9
```

## Comparison: Old vs New Approach

| Feature        | Old (Docker MySQL)           | New (LocalStack RDS)      |
| -------------- | ---------------------------- | ------------------------- |
| AWS Similarity | ❌ Different from production | ✅ Matches AWS RDS        |
| Cost           | Free                         | Free locally              |
| Setup          | Simple                       | Requires Pro token        |
| Testing IaC    | ❌ Can't test Terraform      | ✅ Can test before deploy |
| Port           | 3306                         | 43306                     |
| Container      | Direct MySQL                 | LocalStack + RDS          |

## Environment Variables

The API uses these environment variables (set automatically):

```bash
DB_HOST=localstack        # LocalStack container name
DB_PORT=43306            # LocalStack RDS port
DB_USER=admin
DB_PASSWORD=MySecurePass123!
DB_NAME=shopping_cart_db

# LocalStack-specific
AWS_ENDPOINT_URL=http://localstack:4566
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_REGION=us-west-2
```

## Next Steps

After local development works:

1. Test with the same Terraform configuration
2. Deploy to real AWS with confidence
3. Use the same code for both local and production

## Resources

- [LocalStack Documentation](https://docs.localstack.cloud/)
- [LocalStack RDS](https://docs.localstack.cloud/user-guide/aws/rds/)
- [AWS CLI Local](https://github.com/localstack/awscli-local)
