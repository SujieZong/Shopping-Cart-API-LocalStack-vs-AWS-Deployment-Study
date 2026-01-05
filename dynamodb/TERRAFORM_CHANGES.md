# Terraform Configuration Changes - DynamoDB Deployment

## Summary

Updated the DynamoDB Terraform configuration to add a `dynamodb` prefix to all resources, preventing conflicts with the MySQL deployment.

## Key Changes

### 1. Added Prefix Variable

```hcl
locals {
  prefix = "dynamodb"
}
```

All resource names now use `${local.prefix}-` prefix.

### 2. Updated Network Configuration

| Resource         | Old Value      | New Value      |
| ---------------- | -------------- | -------------- |
| VPC CIDR         | `10.0.0.0/16`  | `10.1.0.0/16`  |
| Public Subnet 1  | `10.0.1.0/24`  | `10.1.1.0/24`  |
| Public Subnet 2  | `10.0.2.0/24`  | `10.1.2.0/24`  |
| Private Subnet 1 | `10.0.10.0/24` | `10.1.10.0/24` |
| Private Subnet 2 | `10.0.11.0/24` | `10.1.11.0/24` |

### 3. Updated Resource Names

| Resource Type    | Old Name                            | New Name                            |
| ---------------- | ----------------------------------- | ----------------------------------- |
| VPC              | `homework7-vpc`                     | `dynamodb-homework7-vpc`            |
| Internet Gateway | `homework7-igw`                     | `dynamodb-homework7-igw`            |
| NAT Gateway      | `homework7-nat`                     | `dynamodb-homework7-nat`            |
| ECS Cluster      | `homework8-cluster`                 | `dynamodb-homework8-cluster`        |
| ALB              | `homework7-alb`                     | `dynamodb-homework7-alb`            |
| Security Groups  | `homework7-alb-*`                   | `dynamodb-homework7-alb-*`          |
| Security Groups  | `homework7-ecs-tasks-*`             | `dynamodb-homework7-ecs-tasks-*`    |
| ECR Repository   | `shopping-cart-api`                 | `dynamodb-shopping-cart-api`        |
| Target Group     | `homework8-shopping-cart-api`       | `dynamodb-shopping-cart-api`        |
| SNS Topic        | `order-processing-events`           | `dynamodb-order-processing-events`  |
| SQS Queue        | `order-processing-queue`            | `dynamodb-order-processing-queue`   |
| CloudWatch Log   | `/ecs/shopping-cart-api`            | `/ecs/dynamodb-shopping-cart-api`   |
| CloudWatch Alarm | `dynamodb-shopping-carts-throttles` | `dynamodb-shopping-carts-throttles` |

### 4. Updated Deployment Scripts

Updated `deploy.sh` to use new resource names:

- ECR Repository: `dynamodb-shopping-cart-api`
- ECS Cluster: `dynamodb-homework8-cluster`
- ALB: `dynamodb-homework7-alb`

## Benefits

✅ **No Conflicts**: DynamoDB and MySQL deployments can coexist in the same AWS account
✅ **Independent Deployment**: Deploy or destroy either stack without affecting the other
✅ **Clear Identification**: Easy to identify which resources belong to which project
✅ **Cost Tracking**: Better cost allocation with distinct resource names

## MySQL Deployment Recommendation

For the MySQL deployment in `mysql/terraform/main.tf`, use:

- Prefix: `mysql`
- VPC CIDR: `10.2.0.0/16`
- Subnet CIDRs: `10.2.1.0/24`, `10.2.2.0/24`, `10.2.10.0/24`, `10.2.11.0/24`

Example:

```hcl
locals {
  prefix = "mysql"
}

resource "aws_vpc" "main" {
  cidr_block = "10.2.0.0/16"
  tags = {
    Name = "${local.prefix}-homework7-vpc"
  }
}
```

## Deployment Instructions

### First Time Deployment

```bash
cd dynamodb/terraform

# Initialize Terraform
terraform init

# Review changes
terraform plan

# Apply configuration
terraform apply

# Deploy application
cd ../shopping-cart-api
./deploy.sh
```

### Update Existing Deployment

If you already have resources deployed without the prefix, you have two options:

#### Option 1: Destroy and Recreate (Recommended for Testing)

```bash
cd dynamodb/terraform

# Destroy old resources
terraform destroy

# Apply new configuration
terraform apply

# Redeploy application
cd ../shopping-cart-api
./deploy.sh
```

#### Option 2: Terraform State Migration (Advanced)

Use `terraform state mv` to rename existing resources in the state file to match the new names. This is complex and error-prone, so only use if you have important data to preserve.

## Notes

- **DynamoDB Table**: The table name `ShoppingCarts` remains unchanged (no prefix) to avoid breaking application references
- **State Files**: Each deployment maintains its own state file in its respective `terraform/` directory
- **AWS Learner Lab**: Both deployments will work with LabRole permissions
- **Cost**: Running both deployments simultaneously will double infrastructure costs

## Verification

After deployment, verify resources are prefixed correctly:

```bash
# Check VPC
aws ec2 describe-vpcs --filters "Name=tag:Name,Values=dynamodb-homework7-vpc"

# Check ECS Cluster
aws ecs describe-clusters --clusters dynamodb-homework8-cluster

# Check ALB
aws elbv2 describe-load-balancers --names dynamodb-homework7-alb

# Check ECR Repository
aws ecr describe-repositories --repository-names dynamodb-shopping-cart-api
```

## Rollback

To remove all DynamoDB resources:

```bash
cd dynamodb/terraform
terraform destroy
```

This will not affect MySQL resources since they have different names.
