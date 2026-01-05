provider "aws" {
  region = "us-west-2"
}

# Prefix for all resources to avoid conflicts
locals {
  prefix = "dynamodb"
}

# Data source to get the existing LabRole
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

# Create VPC as required
resource "aws_vpc" "main" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${local.prefix}-homework7-vpc"
  }
}

# Internet Gateway
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${local.prefix}-homework7-igw"
  }
}

# Public Subnets for ALB
resource "aws_subnet" "public_1" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.1.1.0/24"
  availability_zone       = "us-west-2a"
  map_public_ip_on_launch = true

  tags = {
    Name = "${local.prefix}-homework7-public-1"
  }
}

resource "aws_subnet" "public_2" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.1.2.0/24"
  availability_zone       = "us-west-2b"
  map_public_ip_on_launch = true

  tags = {
    Name = "${local.prefix}-homework7-public-2"
  }
}

# Private Subnets for ECS
resource "aws_subnet" "private_1" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.1.10.0/24"
  availability_zone = "us-west-2a"

  tags = {
    Name = "${local.prefix}-homework7-private-1"
  }
}

resource "aws_subnet" "private_2" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.1.11.0/24"
  availability_zone = "us-west-2b"

  tags = {
    Name = "${local.prefix}-homework7-private-2"
  }
}

# Elastic IP for NAT Gateway
resource "aws_eip" "nat" {
  domain = "vpc"

  tags = {
    Name = "${local.prefix}-homework7-nat-eip"
  }
}

# NAT Gateway in public subnet
resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public_1.id

  tags = {
    Name = "${local.prefix}-homework7-nat"
  }

  depends_on = [aws_internet_gateway.main]
}

# Route table for public subnets
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "${local.prefix}-homework7-public-rt"
  }
}

# Route table for private subnets
resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }

  tags = {
    Name = "${local.prefix}-homework7-private-rt"
  }
}

# Route table associations
resource "aws_route_table_association" "public_1" {
  subnet_id      = aws_subnet.public_1.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "public_2" {
  subnet_id      = aws_subnet.public_2.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "private_1" {
  subnet_id      = aws_subnet.private_1.id
  route_table_id = aws_route_table.private.id
}

resource "aws_route_table_association" "private_2" {
  subnet_id      = aws_subnet.private_2.id
  route_table_id = aws_route_table.private.id
}

# SNS Topic
resource "aws_sns_topic" "order_processing" {
  name = "${local.prefix}-order-processing-events"
}

# SQS Queue with exact requirements
resource "aws_sqs_queue" "order_processing" {
  name                       = "${local.prefix}-order-processing-queue"
  visibility_timeout_seconds = 30         # As required
  message_retention_seconds  = 345600     # 4 days as required
  receive_wait_time_seconds  = 20         # Long polling as required
}

# SNS to SQS subscription
resource "aws_sns_topic_subscription" "order_processing" {
  topic_arn = aws_sns_topic.order_processing.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.order_processing.arn
}

# SQS policy to allow SNS to send messages
resource "aws_sqs_queue_policy" "order_processing" {
  queue_url = aws_sqs_queue.order_processing.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "sns.amazonaws.com"
        }
        Action   = "sqs:SendMessage"
        Resource = aws_sqs_queue.order_processing.arn
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = aws_sns_topic.order_processing.arn
          }
        }
      }
    ]
  })
}

# ECS Cluster
resource "aws_ecs_cluster" "main" {
  name = "${local.prefix}-homework8-cluster"
}

# Security Group for ALB
resource "aws_security_group" "alb" {
  name_prefix = "${local.prefix}-homework7-alb-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# Security Group for ECS tasks
resource "aws_security_group" "ecs_tasks" {
  name_prefix = "${local.prefix}-homework7-ecs-tasks-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# Application Load Balancer - in PUBLIC subnets
resource "aws_lb" "main" {
  name               = "${local.prefix}-homework7-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets           = [aws_subnet.public_1.id, aws_subnet.public_2.id]
}

# ALB Listener
resource "aws_lb_listener" "main" {
  load_balancer_arn = aws_lb.main.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.shopping_cart_api.arn
  }
}

# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "shopping_cart_api" {
  name              = "/ecs/${local.prefix}-shopping-cart-api"
  retention_in_days = 7
}

# ==========================================
# DynamoDB Table for Shopping Carts
# ==========================================

resource "aws_dynamodb_table" "shopping_carts" {
  name           = "ShoppingCarts"
  billing_mode   = "PAY_PER_REQUEST"  # On-demand billing, no capacity planning needed
  hash_key       = "shopping_cart_id"

  # Define the partition key attribute
  attribute {
    name = "shopping_cart_id"
    type = "N"  # Number type as specified
  }

  # Optional: Add Global Secondary Indexes if needed for future queries
  # For example, if you need to query by customer_id later:
  # attribute {
  #   name = "customer_id"
  #   type = "N"
  # }
  # 
  # global_secondary_index {
  #   name            = "CustomerIdIndex"
  #   hash_key        = "customer_id"
  #   projection_type = "ALL"
  # }

  # Point-in-time recovery for data protection (optional but recommended)
  point_in_time_recovery {
    enabled = true
  }

  # Server-side encryption (uses AWS managed keys by default)
  server_side_encryption {
    enabled = true
  }

  # Tags for resource management and cost tracking
  tags = {
    Name        = "ShoppingCarts"
    Environment = "homework8"
    Project     = "ecommerce-api"
    Database    = "dynamodb"
    ManagedBy   = "terraform"
  }

  # Lifecycle rule to prevent accidental deletion in production
  # Remove this for homework/testing environments if needed
  lifecycle {
    prevent_destroy = false  # Set to true in production
  }
}

# ==========================================
# IAM Policy for DynamoDB Access
# ==========================================

# NOTE: In AWS Learner Lab, you cannot create IAM policies or attach them.
# The LabRole already has full DynamoDB permissions by default.
# We're keeping this commented out as documentation only.

# If you were in a regular AWS account, you would use:
# - aws_iam_policy to create a policy
# - aws_iam_role_policy_attachment to attach it to LabRole
#
# But in Learner Lab, LabRole already has these permissions:
# - dynamodb:* (all DynamoDB actions)
# - Full access to all tables in the account
#
# You can verify LabRole permissions in IAM console

# ==========================================
# Update ECS Task Definitions with DynamoDB Environment Variables
# ==========================================

# Update your existing Order API task definition to include DynamoDB table name
# Add this to the environment variables in your order_api task definition:
#
# {
#   name  = "DYNAMODB_TABLE_NAME"
#   value = aws_dynamodb_table.shopping_carts.name
# }
#
# Example of updated task definition (modify your existing one):

resource "aws_ecs_task_definition" "shopping_cart_api" {
  family                   = "shopping-cart-api"
  network_mode            = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                     = "256"
  memory                  = "512"
  
  # Use the existing LabRole for both execution and task role
  execution_role_arn = data.aws_iam_role.lab_role.arn
  task_role_arn      = data.aws_iam_role.lab_role.arn

  container_definitions = jsonencode([
    {
      name  = "shopping-cart-api"
      image = "${aws_ecr_repository.shopping_cart_api.repository_url}:latest"
      
      environment = [
        {
          name  = "DYNAMODB_TABLE_NAME"
          value = aws_dynamodb_table.shopping_carts.name
        },
        {
          name  = "AWS_REGION"
          value = "us-west-2"
        },
        {
          name  = "PORT"
          value = "8080"
        }
      ]

      portMappings = [
        {
          containerPort = 8080
          protocol      = "tcp"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.shopping_cart_api.name
          "awslogs-region"        = "us-west-2"
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])
}

# ECR Repository for Shopping Cart API
resource "aws_ecr_repository" "shopping_cart_api" {
  name                 = "${local.prefix}-shopping-cart-api"
  image_tag_mutability = "MUTABLE"
  force_delete        = true

  image_scanning_configuration {
    scan_on_push = true
  }
}

# Target Group for Shopping Cart API
resource "aws_lb_target_group" "shopping_cart_api" {
  name        = "${local.prefix}-shopping-cart-api"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = aws_vpc.main.id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 5
    timeout             = 10
    interval            = 30
    path                = "/health"
    matcher             = "200"
  }

  deregistration_delay = 30
}

# ECS Service for Shopping Cart API
resource "aws_ecs_service" "shopping_cart_api" {
  name            = "shopping-cart-api"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.shopping_cart_api.arn
  desired_count   = 2  # Run 2 instances for better availability
  launch_type     = "FARGATE"

  health_check_grace_period_seconds = 120

  network_configuration {
    subnets          = [aws_subnet.private_1.id, aws_subnet.private_2.id]
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.shopping_cart_api.arn
    container_name   = "shopping-cart-api"
    container_port   = 8080
  }

  depends_on = [
    aws_lb_listener.main
  ]
}

# ==========================================
# CloudWatch Alarms for DynamoDB Monitoring
# ==========================================

resource "aws_cloudwatch_metric_alarm" "dynamodb_throttles" {
  alarm_name          = "${local.prefix}-shopping-carts-throttles"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name        = "UserErrors"
  namespace          = "AWS/DynamoDB"
  period             = "300"
  statistic          = "Sum"
  threshold          = "10"
  alarm_description  = "This metric monitors DynamoDB throttling"
  treat_missing_data = "notBreaching"

  dimensions = {
    TableName = aws_dynamodb_table.shopping_carts.name
  }
}

# Outputs
output "alb_url" {
  value = "http://${aws_lb.main.dns_name}"
}

output "lab_role_arn" {
  value       = data.aws_iam_role.lab_role.arn
  description = "ARN of the LabRole being used"
}

output "dynamodb_table_name" {
  value       = aws_dynamodb_table.shopping_carts.name
  description = "Name of the DynamoDB ShoppingCarts table"
}

output "dynamodb_table_arn" {
  value       = aws_dynamodb_table.shopping_carts.arn
  description = "ARN of the DynamoDB ShoppingCarts table"
}

output "dynamodb_table_id" {
  value       = aws_dynamodb_table.shopping_carts.id
  description = "ID of the DynamoDB ShoppingCarts table"
}

output "shopping_cart_api_url" {
  value       = "http://${aws_lb.main.dns_name}/shopping-carts"
  description = "URL for the Shopping Cart API"
}

output "ecr_shopping_cart_api_url" {
  value       = aws_ecr_repository.shopping_cart_api.repository_url
  description = "ECR repository URL for Shopping Cart API"
}

# Output for verification
output "lab_role_policies" {
  value       = data.aws_iam_role.lab_role.arn
  description = "LabRole ARN with DynamoDB permissions attached"
}