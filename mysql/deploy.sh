#!/bin/bash
# deploy.sh - 简化的一键部署脚本（适合Live Demo）

set -e

echo "=========================================="
echo "🚀 One-Click Deployment for Shopping Cart API"
echo "=========================================="

# 配置
AWS_REGION="us-west-2"
PROJECT_NAME="hw8-mysql"
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null)

if [ -z "$AWS_ACCOUNT_ID" ]; then
    echo "❌ Error: Not logged into AWS"
    echo "Please run: aws configure"
    exit 1
fi

ECR_REPO="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${PROJECT_NAME}-app"

echo "✅ AWS Account: $AWS_ACCOUNT_ID"
echo "✅ Region: $AWS_REGION"

# Step 1: 部署基础设施
echo -e "\n📦 Step 1: Deploying Infrastructure with Terraform..."
cd terraform

# 初始化Terraform（如果需要）
if [ ! -d ".terraform" ]; then
    terraform init
fi

# 检查是否需要部署
NEEDS_APPLY=false
if ! terraform show 2>/dev/null | grep -q "aws_vpc.main"; then
    echo "Infrastructure not found, deploying..."
    NEEDS_APPLY=true
else
    echo "Infrastructure exists, checking for changes..."
    if terraform plan -detailed-exitcode > /dev/null 2>&1; then
        echo "No infrastructure changes needed."
    else
        echo "Infrastructure changes detected, applying..."
        NEEDS_APPLY=true
    fi
fi

if [ "$NEEDS_APPLY" = true ]; then
    terraform apply -auto-approve
    echo "Waiting for RDS to be fully ready (60s for new instance)..."
    sleep 60
fi

# 获取输出
ALB_URL=$(terraform output -raw alb_url)
RDS_ENDPOINT=$(terraform output -raw rds_endpoint)
ECS_CLUSTER=$(terraform output -raw ecs_cluster_name)
ECS_SERVICE=$(terraform output -raw ecs_service_name)

cd ..

echo "Infrastructure ready:"
echo "  ALB URL: ${ALB_URL}"
echo "  RDS Endpoint: ${RDS_ENDPOINT}"

# Step 2: 构建Docker镜像
echo -e "\n🐳 Step 2: Building Docker Image..."
cd src
docker build --platform linux/amd64 -t ${PROJECT_NAME}-app .
cd ..
echo "✅ Docker image built successfully"

# Step 3: 登录ECR
echo -e "\n🔐 Step 3: Authenticating with ECR..."
aws ecr get-login-password --region ${AWS_REGION} | \
    docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Step 4: 推送镜像
echo -e "\n⬆️ Step 4: Pushing Image to ECR..."
docker tag ${PROJECT_NAME}-app:latest ${ECR_REPO}:latest
docker push ${ECR_REPO}:latest
echo "✅ Image pushed to ECR"

# Step 5: 更新ECS服务
echo -e "\n♻️ Step 5: Updating ECS Service..."
aws ecs update-service \
    --cluster ${ECS_CLUSTER} \
    --service ${ECS_SERVICE} \
    --force-new-deployment \
    --region ${AWS_REGION} \
    --output json > /dev/null

echo "⏳ Waiting for service to stabilize..."

# 等待服务稳定（显示进度）
MAX_ATTEMPTS=30
ATTEMPT=0
STABLE=false

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    
    # 获取服务状态
    RUNNING_COUNT=$(aws ecs describe-services \
        --cluster ${ECS_CLUSTER} \
        --services ${ECS_SERVICE} \
        --region ${AWS_REGION} \
        --query 'services[0].runningCount' \
        --output text 2>/dev/null || echo "0")
    
    DESIRED_COUNT=$(aws ecs describe-services \
        --cluster ${ECS_CLUSTER} \
        --services ${ECS_SERVICE} \
        --region ${AWS_REGION} \
        --query 'services[0].desiredCount' \
        --output text 2>/dev/null || echo "2")
    
    echo -ne "\r⏳ Tasks running: ${RUNNING_COUNT}/${DESIRED_COUNT} (Attempt ${ATTEMPT}/${MAX_ATTEMPTS})"
    
    if [ "$RUNNING_COUNT" -eq "$DESIRED_COUNT" ] && [ "$RUNNING_COUNT" -gt "0" ]; then
        echo -e "\n✅ Service is stable with $RUNNING_COUNT tasks running!"
        STABLE=true
        break
    fi
    
    sleep 10
done

if [ "$STABLE" = false ]; then
    echo -e "\n⚠️ Service deployment timeout - checking health anyway..."
fi

# Step 6: 验证部署
echo -e "\n🔍 Step 6: Verifying Deployment..."
sleep 10

# 多次尝试健康检查
MAX_HEALTH_ATTEMPTS=10
HEALTH_ATTEMPT=0
HEALTH_SUCCESS=false

while [ $HEALTH_ATTEMPT -lt $MAX_HEALTH_ATTEMPTS ]; do
    HEALTH_ATTEMPT=$((HEALTH_ATTEMPT + 1))
    
    echo -n "Health check attempt ${HEALTH_ATTEMPT}/${MAX_HEALTH_ATTEMPTS}: "
    HEALTH_CHECK=$(curl -s -o /dev/null -w "%{http_code}" ${ALB_URL}/health 2>/dev/null || echo "000")
    echo "HTTP ${HEALTH_CHECK}"
    
    if [ "$HEALTH_CHECK" = "200" ]; then
        echo "✅ Health check passed!"
        HEALTH_SUCCESS=true
        break
    fi
    
    sleep 5
done

# Step 7: 测试API功能
if [ "$HEALTH_SUCCESS" = true ]; then
    echo -e "\n🧪 Testing API functionality..."
    
    # 测试创建购物车
    echo "Testing cart creation..."
    RESPONSE=$(curl -s -X POST ${ALB_URL}/shopping-carts \
        -H "Content-Type: application/json" \
        -d '{"customer_id": 1000}' 2>/dev/null || echo "Failed")
    
    if echo "$RESPONSE" | grep -q "shopping_cart_id"; then
        echo "✅ API test successful! Response: $RESPONSE"
        
        # 提取cart ID并测试其他功能
        CART_ID=$(echo "$RESPONSE" | grep -o '"shopping_cart_id":[0-9]*' | grep -o '[0-9]*')
        
        if [ ! -z "$CART_ID" ]; then
            # 测试添加商品
            echo "Testing add items to cart $CART_ID..."
            ADD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
                -X POST ${ALB_URL}/shopping-carts/${CART_ID}/items \
                -H "Content-Type: application/json" \
                -d '{"product_id": 101, "quantity": 2}')
            
            if [ "$ADD_STATUS" = "204" ]; then
                echo "✅ Add items successful (HTTP 204)"
                
                # 测试获取购物车
                echo "Testing get cart..."
                GET_RESPONSE=$(curl -s ${ALB_URL}/shopping-carts/${CART_ID})
                if echo "$GET_RESPONSE" | grep -q "product_id"; then
                    echo "✅ Get cart successful"
                fi
            fi
        fi
    else
        echo "⚠️ API test returned unexpected response: $RESPONSE"
    fi
else
    echo "⚠️ Health check failed after ${MAX_HEALTH_ATTEMPTS} attempts"
    echo "Check CloudWatch logs for details:"
    echo "https://console.aws.amazon.com/cloudwatch/home?region=${AWS_REGION}#logsV2:log-groups/log-group/\$252Fecs\$252F${PROJECT_NAME}"
fi

# 显示最终信息
echo ""
echo "=========================================="
if [ "$HEALTH_SUCCESS" = true ]; then
    echo "🎉 Deployment Successful!"
else
    echo "⚠️ Deployment Complete (with warnings)"
fi
echo "=========================================="
echo ""
echo "📌 API Endpoint: ${ALB_URL}"
echo ""
echo "🧪 Quick Test Commands:"
echo ""
echo "# Set API URL:"
echo "export API_URL=${ALB_URL}"
echo ""
echo "# Health check:"
echo "curl \$API_URL/health"
echo ""
echo "# Create cart:"
echo "curl -X POST \$API_URL/shopping-carts \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"customer_id\": 1000}'"
echo ""
echo "# Run performance test:"
echo "./test.sh"
echo ""
echo "📊 AWS Console Links:"
echo "  ECS Tasks: https://console.aws.amazon.com/ecs/v2/clusters/${ECS_CLUSTER}/services/${ECS_SERVICE}/tasks"
echo "  CloudWatch Logs: https://console.aws.amazon.com/cloudwatch/home?region=${AWS_REGION}#logsV2:log-groups"
echo ""

# 如果部署成功，保存配置供后续使用
if [ "$HEALTH_SUCCESS" = true ]; then
    echo "# Auto-generated deployment config" > .deployment_config
    echo "export API_URL=${ALB_URL}" >> .deployment_config
    echo "export ECS_CLUSTER=${ECS_CLUSTER}" >> .deployment_config
    echo "export ECS_SERVICE=${ECS_SERVICE}" >> .deployment_config
    echo "export AWS_REGION=${AWS_REGION}" >> .deployment_config
    echo "✅ Deployment config saved to .deployment_config"
fi

echo ""
echo "🧹 Cleanup command (after demo):"
echo "cd terraform && terraform destroy -auto-approve && cd .."