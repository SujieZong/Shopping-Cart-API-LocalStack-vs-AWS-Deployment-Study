#!/bin/bash
set -e

# Infrastructure Setup Script
# Run this BEFORE deploying the application

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== Setting up AWS Infrastructure with Terraform ===${NC}"

# Check if we're in AWS Learner Lab
echo -e "\n${BLUE}Checking AWS environment...${NC}"
AWS_ACCOUNT=$(aws sts get-caller-identity --query Account --output text 2>/dev/null || echo "")
if [ -z "$AWS_ACCOUNT" ]; then
    echo -e "${YELLOW}Warning: AWS credentials not configured${NC}"
    echo -e "Please configure AWS CLI with your credentials"
    exit 1
fi

echo -e "${GREEN}✓ AWS Account ID: ${AWS_ACCOUNT}${NC}"

# Note about LabRole permissions
echo -e "\n${BLUE}Note: This script works with AWS Learner Lab LabRole${NC}"
echo -e "LabRole already has DynamoDB permissions - no IAM policy creation needed"

cd ../terraform

# Step 1: Initialize Terraform
echo -e "\n${GREEN}Step 1: Initializing Terraform...${NC}"
terraform init

# Step 2: Validate configuration
echo -e "\n${GREEN}Step 2: Validating Terraform configuration...${NC}"
terraform validate

# Step 3: Plan infrastructure changes
echo -e "\n${GREEN}Step 3: Planning infrastructure changes...${NC}"
terraform plan -out=tfplan

# Step 4: Ask for confirmation
echo -e "\n${YELLOW}Ready to apply changes. This will create:${NC}"
echo -e "  - DynamoDB table (ShoppingCarts)"
echo -e "  - ECR repository (shopping-cart-api)"
echo -e "  - ECS cluster, task definition, and service"
echo -e "  - Application Load Balancer"
echo -e "  - VPC, subnets, security groups"
echo -e "  - IAM roles and policies"
echo ""
read -p "Do you want to proceed? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo -e "${YELLOW}Deployment cancelled${NC}"
    exit 0
fi

# Step 5: Apply Terraform configuration
echo -e "\n${GREEN}Step 4: Applying Terraform configuration...${NC}"
terraform apply tfplan

# Step 6: Show outputs
echo -e "\n${GREEN}=== Infrastructure Created Successfully! ===${NC}"
echo -e "\n${GREEN}Terraform Outputs:${NC}"
terraform output

echo -e "\n${YELLOW}Next steps:${NC}"
echo -e "  1. cd ../shopping-cart-api"
echo -e "  2. ./deploy.sh"
echo -e "  3. Use Postman or curl to test the API"
