#!/bin/bash
set -e

# AWS Deployment Script for Shopping Cart API
# This script builds, pushes Docker image to ECR, and deploys to ECS

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Shopping Cart API Deployment Script ===${NC}"

# Configuration
AWS_REGION=${AWS_REGION:-us-west-2}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REPOSITORY_NAME="dynamodb-shopping-cart-api"  # ✅ Matches Terraform
IMAGE_TAG=${IMAGE_TAG:-latest}

echo -e "${YELLOW}AWS Account ID: ${AWS_ACCOUNT_ID}${NC}"
echo -e "${YELLOW}Region: ${AWS_REGION}${NC}"
echo -e "${YELLOW}ECR Repository: ${ECR_REPOSITORY_NAME}${NC}"

# Step 1: Build Docker image for linux/amd64 (required for ECS Fargate)
echo -e "\n${GREEN}Step 1: Building Docker image for linux/amd64...${NC}"
docker build --platform linux/amd64 -t ${ECR_REPOSITORY_NAME}:${IMAGE_TAG} .

# Step 2: Get ECR repository URI (should be created by Terraform)
echo -e "\n${GREEN}Step 2: Getting ECR repository URI...${NC}"
ECR_URI="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY_NAME}"
echo -e "${YELLOW}ECR URI: ${ECR_URI}${NC}"

# Step 3: Authenticate Docker to ECR
echo -e "\n${GREEN}Step 3: Authenticating Docker to ECR...${NC}"
aws ecr get-login-password --region ${AWS_REGION} | \
    docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Step 4: Tag image for ECR
echo -e "\n${GREEN}Step 4: Tagging image for ECR...${NC}"
docker tag ${ECR_REPOSITORY_NAME}:${IMAGE_TAG} ${ECR_URI}:${IMAGE_TAG}

# Step 5: Push image to ECR
echo -e "\n${GREEN}Step 5: Pushing image to ECR...${NC}"
docker push ${ECR_URI}:${IMAGE_TAG}

# Step 6: Update ECS service (force new deployment)
echo -e "\n${GREEN}Step 6: Updating ECS service...${NC}"
CLUSTER_NAME="dynamodb-homework8-cluster"
SERVICE_NAME="shopping-cart-api"

aws ecs update-service \
    --cluster ${CLUSTER_NAME} \
    --service ${SERVICE_NAME} \
    --force-new-deployment \
    --region ${AWS_REGION} > /dev/null

echo -e "\n${GREEN}✓ Deployment initiated successfully!${NC}"
echo -e "${YELLOW}Waiting for service to stabilize...${NC}"

# Step 7: Wait for service to become stable
aws ecs wait services-stable \
    --cluster ${CLUSTER_NAME} \
    --services ${SERVICE_NAME} \
    --region ${AWS_REGION}

echo -e "\n${GREEN}✓ Service is now stable!${NC}"

# Step 8: Get the ALB URL
echo -e "\n${GREEN}Step 8: Getting Application Load Balancer URL...${NC}"
ALB_URL=$(aws elbv2 describe-load-balancers \
    --names dynamodb-homework7-alb \
    --region ${AWS_REGION} \
    --query 'LoadBalancers[0].DNSName' \
    --output text 2>/dev/null || echo "")

if [ -n "$ALB_URL" ]; then
    echo -e "${GREEN}✓ Deployment complete!${NC}"
    echo -e "\n${YELLOW}=== API Endpoint ===${NC}"
    echo -e "Base URL: ${GREEN}http://${ALB_URL}${NC}"
    echo -e "Shopping Carts: ${GREEN}http://${ALB_URL}/shopping-carts${NC}"
    echo -e "Health Check: ${GREEN}http://${ALB_URL}/health${NC}"
    
    echo -e "\n${YELLOW}=== Test Commands ===${NC}"
    echo -e "Health check:"
    echo -e "  curl http://${ALB_URL}/health"
    echo -e "\nCreate cart:"
    echo -e "  curl -X POST http://${ALB_URL}/shopping-carts \\"
    echo -e "    -H \"Content-Type: application/json\" \\"
    echo -e "    -d '{\"customer_id\": 12345}'"
else
    echo -e "${RED}Warning: Could not retrieve ALB URL${NC}"
    echo -e "${YELLOW}Check AWS Console for the endpoint${NC}"
fi

echo -e "\n${GREEN}=== Deployment Summary ===${NC}"
echo -e "Image: ${ECR_URI}:${IMAGE_TAG}"
echo -e "Cluster: ${CLUSTER_NAME}"
echo -e "Service: ${SERVICE_NAME}"
echo -e "Region: ${AWS_REGION}"