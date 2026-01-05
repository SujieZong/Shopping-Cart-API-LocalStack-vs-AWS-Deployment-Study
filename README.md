# Shopping Cart API: LocalStack vs AWS Deployment Study

A comprehensive distributed systems project comparing local development (LocalStack) and production cloud (AWS) deployments of a shopping cart REST API. This project demonstrates practical experience with containerization, cloud infrastructure, database management, and performance analysis.

**Course:** CS 6650 - Distributed Systems Engineering  
**Author:** Sujie Zong  
**Date:** December 2025

---

## 📋 Project Overview

This project implements a shopping cart REST API in **Go** with support for two database backends (**MySQL** and **DynamoDB**) deployed across two distinct environments:

- **LocalStack** (local Docker-based development)
- **AWS** (production cloud infrastructure)

### Key Deliverables

- ✅ Fully functional REST API with CRUD operations
- ✅ Dual database implementations (MySQL & DynamoDB)
- ✅ LocalStack setup for rapid local development
- ✅ AWS infrastructure provisioning with Terraform
- ✅ Comprehensive performance testing suite
- ✅ Detailed analysis report comparing deployments

---

## 🎯 High-Level Architecture

```
┌────────────────────────────────────────────────────────────┐
│                     Client Layer                           │
│                  (HTTP/REST Clients)                       │
└────────────────────┬─────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼────────┐       ┌────────▼────────┐
│  LocalStack    │       │   AWS Cloud     │
│  (Development) │       │  (Production)   │
│                │       │                 │
│ ┌──────────┐   │       │ ┌──────────┐    │
│ │  Go API  │   │       │ │   Go API │    │
│ │  (8080)  │   │       │ │  (ECS)   │    │
│ └─────┬────┘   │       │ └─────┬────┘    │
│       │        │       │       │         │
│  ┌────▼────┐   │       │   ┌───▼───┐     │
│  │ MySQL   │◄──┼───┐   │   │  RDS  │     │
│  │ DynamoDB│   │   │   │   │ DynoDB│     │
│  └─────────┘   │   │   │   └───────┘     │
│                │   │   │                 │
└────────────────┘   │   └─────────────────┘
                     │
          Performance Testing
          & Metrics Collection
```

---

## 📂 Directory Structure

```
final/
├── README.md (this file)
├── DEPLOYMENT_ANALYSIS_REPORT.md          # Main analysis report
├── INTERVIEW_QUICK_REFERENCE.md           # Interview prep guide
│
├── dynamodb/                               # DynamoDB implementation
│   ├── README.md
│   ├── SCHEMA_DESIGN.md
│   ├── DEPLOYMENT_GUIDE.md
│   ├── TERRAFORM_CHANGES.md
│   ├── shopping-cart-api/                 # Go API source code
│   │   ├── main.go
│   │   ├── handlers.go
│   │   ├── models.go
│   │   ├── repository.go
│   │   ├── Dockerfile
│   │   ├── docker-compose.localstack.yml
│   │   ├── go.mod / go.sum
│   │   ├── setup-infrastructure.sh         # AWS provisioning
│   │   ├── setup-localstack.sh             # LocalStack setup
│   │   ├── run-local.sh                    # Quick start
│   │   └── QUICKSTART.md
│   ├── terraform/                         # AWS infrastructure
│   │   ├── main.tf
│   │   └── terraform.tfstate
│   └── tests/                             # Performance tests
│       ├── *.go                           # Go test files
│       └── *.json                         # Test results
│
├── mysql/                                 # MySQL implementation
│   ├── README.md
│   ├── DEPLOYMENT_GUIDE.md
│   ├── MySQL_Implementation_Notes.md
│   ├── src/                               # Go API source code
│   ├── terraform/                         # AWS infrastructure
│   ├── tests/                             # Performance tests
│   ├── docker-compose.yml                 # Local MySQL setup
│   ├── run-local.sh
│   ├── setup-localstack.sh
│   └── deploy.sh
│
└── analysis/                              # Performance analysis
    ├── README.md
    ├── analyze_results.py
    ├── generate_charts.py
    ├── combined_results.json
    ├── dynamodb_test_results.json
    ├── mysql_test_results.json
    └── charts/                            # Generated visualizations
        ├── baseline_performance_comparison.png
        ├── load_scaling_comparison.png
        ├── cold_start_comparison.png
        ├── cost_comparison.png
        └── latency_heatmap.png
```

---

## 🚀 Quick Start

### Option 1: LocalStack (Recommended for Development)

**LocalStack** provides a fully local, fast development environment with zero AWS costs.

```bash
# DynamoDB version
cd dynamodb/shopping-cart-api
./setup-localstack.sh
./run-local.sh

# MySQL version
cd mysql
./run-local.sh
```

**Test the API:**

```bash
curl -X POST http://localhost:8080/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 12345}'
```

📖 See [dynamodb/QUICKSTART.md](dynamodb/QUICKSTART.md) or [mysql/README.md](mysql/README.md)

### Option 2: AWS Deployment (Production)

**AWS** provides enterprise-grade reliability and scalability. Requires AWS credentials and Terraform.

```bash
# DynamoDB on AWS
cd dynamodb/shopping-cart-api
./setup-infrastructure.sh
./deploy.sh

# MySQL on AWS
cd mysql
terraform apply
./deploy.sh
```

📖 See [dynamodb/DEPLOYMENT_GUIDE.md](dynamodb/DEPLOYMENT_GUIDE.md)

---

## 📊 API Specification

### Endpoints

| Method | Endpoint                     | Purpose           | Response                          |
| ------ | ---------------------------- | ----------------- | --------------------------------- |
| `POST` | `/shopping-carts`            | Create new cart   | `201` + `{shopping_cart_id: int}` |
| `POST` | `/shopping-carts/{id}/items` | Add item to cart  | `204` No Content                  |
| `GET`  | `/shopping-carts/{id}`       | Get cart contents | `200` + cart JSON                 |
| `GET`  | `/health`                    | Health check      | `200` + `{status: "ok"}`          |

### Request/Response Examples

**Create Shopping Cart:**

```bash
POST /shopping-carts
Content-Type: application/json

{"customer_id": 12345}

# Response (201)
{"shopping_cart_id": 67890}
```

**Add Item to Cart:**

```bash
POST /shopping-carts/67890/items
Content-Type: application/json

{"product_id": 100, "quantity": 2}

# Response (204) - No Content
```

**Get Cart:**

```bash
GET /shopping-carts/67890

# Response (200)
{
  "shopping_cart_id": 67890,
  "customer_id": 12345,
  "items": [
    {"product_id": 100, "quantity": 2},
    {"product_id": 200, "quantity": 1}
  ]
}
```

---

## 🗄️ Database Schemas

### MySQL Implementation

**Shopping Carts Table:**

```sql
CREATE TABLE shopping_carts (
  shopping_cart_id INT PRIMARY KEY AUTO_INCREMENT,
  customer_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE cart_items (
  item_id INT PRIMARY KEY AUTO_INCREMENT,
  shopping_cart_id INT NOT NULL REFERENCES shopping_carts(shopping_cart_id),
  product_id INT NOT NULL,
  quantity INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### DynamoDB Implementation

**ShoppingCarts Table:**

- **Partition Key:** `shopping_cart_id` (Number)
- **Attributes:**
  - `customer_id` (Number)
  - `items` (Map of product_id → quantity)
  - `created_at` (String - ISO 8601)

📖 Full schema documentation: [dynamodb/SCHEMA_DESIGN.md](dynamodb/SCHEMA_DESIGN.md)

---

## 🧪 Testing & Performance Analysis

### Test Suite

The project includes comprehensive testing across 5 categories:

1. **Baseline Performance** (150 test runs)

   - Measures: average latency, P50, P95, P99 percentiles
   - Validates: functional correctness and latency profiles

2. **Load Scaling** (3 load levels × 3 operations)

   - Tests: 50, 100, 167 concurrent users
   - Validates: throughput under load, resource scaling

3. **Cold Start Analysis**

   - Infrastructure provisioning time
   - Container startup overhead

4. **Consistency Testing**

   - Database consistency under concurrent operations
   - Data integrity verification

5. **Failure Mode Testing**
   - Invalid requests handling
   - Missing resources gracefully
   - Network error resilience

### Running Tests

```bash
# DynamoDB tests
cd dynamodb/tests
go test -v -timeout 10m

# MySQL tests
cd mysql/tests
go test -v -timeout 10m

# Load testing with Locust
cd dynamodb
python3 -m locust -f locustfile.py --host=http://localhost:8080
```

### Analyzing Results

```bash
cd analysis
python3 generate_charts.py
# Output: performance charts in analysis/charts/
```

---

## 📈 Key Findings

### Performance Metrics

| Metric                | LocalStack  | AWS         | Difference              |
| --------------------- | ----------- | ----------- | ----------------------- |
| **Avg Response Time** | 8.45ms      | 44.07ms     | AWS 5.2× slower         |
| **P99 Latency**       | 518ms       | 78ms        | AWS 6.6× faster         |
| **Throughput**        | 432 ops/sec | 238 ops/sec | LocalStack 1.8× higher  |
| **Cold Start**        | 0.006s      | 30.4s       | AWS 5,000× slower       |
| **Monthly Cost**      | $0          | ~$50        | LocalStack 100% savings |
| **Consistency**       | 100%        | 100%        | Equal reliability       |

### Recommendations

**Development (90% of work):**

- Use **LocalStack** for rapid iteration
- 5-20× faster development cycles
- Zero infrastructure costs
- Full feature parity with AWS

**Production (10% validation):**

- Deploy on **AWS** for reliability
- 99.9%+ uptime SLA
- Better P99 latency (78ms vs 518ms)
- Managed backups and disaster recovery

**Hybrid Approach Benefits:**

- ✅ Achieve 85% cost savings vs AWS-only
- ✅ Maintain production-grade quality
- ✅ Rapid prototyping and debugging
- ✅ Minimal dependency on AWS during development

---

## 🛠️ Technology Stack

### Languages & Frameworks

- **Language:** Go 1.21
- **Web Framework:** Gin Web Framework
- **Database Drivers:**
  - MySQL: Database/SQL + Go-MySQL-Driver
  - DynamoDB: AWS SDK for Go v2

### Infrastructure & DevOps

- **Containerization:** Docker
- **Orchestration:** Docker Compose (local), ECS Fargate (AWS)
- **Infrastructure as Code:** Terraform
- **Local Development:** LocalStack

### Testing & Monitoring

- **Load Testing:** Locust (Python)
- **Performance Testing:** Go Testing Framework
- **Analysis:** Python (matplotlib, pandas, numpy)

---

## 📚 Documentation

### Project Documentation

| Document                                                                   | Purpose                                                   |
| -------------------------------------------------------------------------- | --------------------------------------------------------- |
| [DEPLOYMENT_ANALYSIS_REPORT.md](DEPLOYMENT_ANALYSIS_REPORT.md)             | Comprehensive 5-page analysis comparing LocalStack vs AWS |
| [INTERVIEW_QUICK_REFERENCE.md](INTERVIEW_QUICK_REFERENCE.md)               | Quick reference for technical interviews                  |
| [dynamodb/SCHEMA_DESIGN.md](dynamodb/SCHEMA_DESIGN.md)                     | DynamoDB schema design rationale                          |
| [dynamodb/TERRAFORM_CHANGES.md](dynamodb/TERRAFORM_CHANGES.md)             | AWS infrastructure setup details                          |
| [mysql/MySQL_Implementation_Notes.md](mysql/MySQL_Implementation_Notes.md) | MySQL-specific implementation details                     |

### Quick Start Guides

- [dynamodb/QUICKSTART.md](dynamodb/QUICKSTART.md) - Get DynamoDB API running in 2 minutes
- [dynamodb/LOCALSTACK_QUICKSTART.md](dynamodb/LOCALSTACK_QUICKSTART.md) - LocalStack detailed setup
- [mysql/README.md](mysql/README.md) - MySQL quick start

### Deployment Guides

- [dynamodb/DEPLOYMENT_GUIDE.md](dynamodb/DEPLOYMENT_GUIDE.md) - Deploy DynamoDB API to AWS
- [mysql/DEPLOYMENT_GUIDE.md](mysql/DEPLOYMENT_GUIDE.md) - Deploy MySQL API to AWS

---

## 🔧 Development Workflow

### Local Setup (First Time)

```bash
# Clone and navigate
cd final

# Start LocalStack DynamoDB version
cd dynamodb/shopping-cart-api
./setup-localstack.sh
./run-local.sh

# In another terminal, run tests
cd ../tests
go test -v -timeout 10m
```

### Adding New Endpoints

1. Define handler in `handlers.go`
2. Add repository method in `repository.go`
3. Register route in `main.go`
4. Add tests in `tests/` directory
5. Run tests: `go test -v`

### Database Migrations

**MySQL:**

```bash
# Update schema in setup-local-db.sh
./setup-local-db.sh
```

**DynamoDB:**

```bash
# Update table definition in setup-localstack.sh or Terraform
./setup-localstack.sh
```

---

## 📋 Key Implementation Details

### Go API Features

- **Error Handling:** Comprehensive HTTP error responses with validation
- **CORS:** Enabled for cross-origin requests
- **Logging:** Structured logging for debugging and monitoring
- **Configuration:** Environment variable support for flexibility
- **Health Checks:** Built-in `/health` endpoint

### MySQL Specifics

- Supports AUTO_INCREMENT cart IDs
- Transaction support for multi-operation updates
- Index optimization for query performance

### DynamoDB Specifics

- Efficient NoSQL single-table design
- Map type for flexible item storage
- Optional consistent reads for strong consistency
- Optimized for both LocalStack and AWS

---

## 🤝 Contributing

This is an academic project. To extend or modify:

1. **Database Changes:** Update schema files, then migrations
2. **API Changes:** Update OpenAPI spec, handlers, tests
3. **Infrastructure:** Modify Terraform files, then apply
4. **Performance Testing:** Add test cases in `tests/` directory
5. **Analysis:** Add analysis scripts in `analysis/` directory

---

## 📝 License

This project is created as coursework for CS 6650 - Distributed Systems Engineering at Northeastern University.


---

## 📌 Quick Reference

### Common Commands

```bash
# LocalStack development
cd dynamodb/shopping-cart-api
./setup-localstack.sh && ./run-local.sh

# Run tests
cd ../tests && go test -v -timeout 10m

# Generate performance charts
cd ../../analysis && python3 generate_charts.py

# Deploy to AWS
cd ../dynamodb/shopping-cart-api
./setup-infrastructure.sh && ./deploy.sh

# Check application logs
docker logs shopping-cart-api

# Clean up Docker resources
docker-compose down -v
```

### Useful Links

- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
- [Gin Web Framework](https://gin-gonic.com/)
- [LocalStack Documentation](https://docs.localstack.cloud/)
- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)


---

**Last Updated:** January 4, 2026
