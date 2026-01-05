# Shopping Cart API - MySQL Quick Start

## 🚀 Quick Start Guide

Choose your deployment method:

## 📍 Local Development (Fastest)

Perfect for development and testing on your local machine.

### Prerequisites

- Docker Desktop installed and running

### Start in 1 Command

```bash
cd mysql
./run-local.sh
```

**That's it!** The script will:

- ✅ Start MySQL database
- ✅ Start API server
- ✅ Run health checks
- ✅ Test all endpoints

**Access your API at**: `http://localhost:3000`

### Stop Services

```bash
docker-compose down
```

---

## ☁️ AWS Deployment

Deploy to production on AWS.

### Prerequisites

- AWS Account configured
- AWS CLI installed
- Terraform installed
- Docker installed

### Deploy in 1 Command

```bash
cd mysql
./deploy.sh
```

The script will:

- 📦 Create AWS infrastructure (VPC, RDS, ECS, ALB)
- 🐳 Build and push Docker image
- 🚀 Deploy your application
- ✅ Verify deployment

**Get your API URL**:

```bash
cd terraform
terraform output alb_url
```

### Cleanup

```bash
cd mysql
./cleanup.sh
```

---

## 📖 Full Documentation

For detailed instructions, troubleshooting, and architecture information, see:

**[DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)**

---

## 🧪 Quick API Test

```bash
# Health check
curl http://localhost:3000/health

# Create cart
curl -X POST http://localhost:3000/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 123}'

# Add item (replace {id} with cart ID from above)
curl -X POST http://localhost:3000/shopping-carts/{id}/items \
  -H "Content-Type: application/json" \
  -d '{"product_id": 456, "quantity": 2}'

# Get cart
curl http://localhost:3000/shopping-carts/{id}
```

---

## 📁 Project Structure

```
mysql/
├── docker-compose.yml          # Local development setup
├── run-local.sh               # Quick start script for local
├── deploy.sh                  # AWS deployment script
├── cleanup.sh                 # AWS cleanup script
├── DEPLOYMENT_GUIDE.md        # Complete documentation
├── README.md                  # This file
├── src/
│   ├── main.go               # Application code
│   ├── Dockerfile            # Container image
│   └── db/
│       └── setup.sql         # Database schema
└── terraform/
    └── main.tf               # AWS infrastructure
```

---

## 🛠️ Common Commands

### Local Development

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Reset everything
docker-compose down -v && docker-compose up -d
```

### AWS Deployment

```bash
# Deploy/Update
./deploy.sh

# Check status
cd terraform
terraform output

# View logs
aws logs tail /ecs/hw8-mysql --follow

# Cleanup
./cleanup.sh
```

---

## 🆘 Need Help?

1. **Check health endpoint**: `curl http://localhost:3000/health`
2. **View logs**: `docker-compose logs -f api`
3. **See full guide**: [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)
4. **Database connection**:
   ```bash
   docker exec -it shopping-cart-mysql mysql -u admin -pMySecurePass123! shopping_cart_db
   ```

---

## 🎯 What's Included

✅ **Local Development** with Docker Compose  
✅ **AWS Production** deployment with Terraform  
✅ **Auto-scaling** ECS Fargate service (2-4 tasks)  
✅ **MySQL 8.0** database (local Docker or AWS RDS)  
✅ **Load Balancer** for high availability  
✅ **Health checks** and monitoring  
✅ **Automated tests** and deployment scripts

---

**Ready to start?** Run `./run-local.sh` for local development or `./deploy.sh` for AWS!
