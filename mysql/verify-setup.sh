#!/bin/bash
# verify-setup.sh - Verify that everything is set up correctly

set -e

echo "=========================================="
echo "🔍 MySQL Implementation Setup Verification"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

ERRORS=0

# Check Docker
echo "1. Checking Docker..."
if docker info > /dev/null 2>&1; then
    echo -e "   ${GREEN}✅ Docker is installed and running${NC}"
else
    echo -e "   ${RED}❌ Docker is not running${NC}"
    echo "      Please start Docker Desktop"
    ERRORS=$((ERRORS + 1))
fi

# Check required files
echo ""
echo "2. Checking required files..."

FILES=(
    "docker-compose.yml"
    "run-local.sh"
    "setup-local-db.sh"
    "deploy.sh"
    "cleanup.sh"
    "DEPLOYMENT_GUIDE.md"
    "README.md"
    "src/main.go"
    "src/Dockerfile"
    "src/db/setup.sql"
    "terraform/main.tf"
)

for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "   ${GREEN}✅${NC} $file"
    else
        echo -e "   ${RED}❌${NC} $file (missing)"
        ERRORS=$((ERRORS + 1))
    fi
done

# Check script permissions
echo ""
echo "3. Checking script permissions..."

SCRIPTS=(
    "run-local.sh"
    "setup-local-db.sh"
    "deploy.sh"
    "cleanup.sh"
)

for script in "${SCRIPTS[@]}"; do
    if [ -x "$script" ]; then
        echo -e "   ${GREEN}✅${NC} $script is executable"
    else
        echo -e "   ${YELLOW}⚠️${NC}  $script needs execute permission"
        chmod +x "$script"
        echo -e "      Fixed: chmod +x $script"
    fi
done

# Check Go files
echo ""
echo "4. Checking Go code..."
if [ -f "src/go.mod" ]; then
    echo -e "   ${GREEN}✅${NC} go.mod exists"
    
    cd src
    if go mod verify > /dev/null 2>&1; then
        echo -e "   ${GREEN}✅${NC} Go dependencies are valid"
    else
        echo -e "   ${YELLOW}⚠️${NC}  Running go mod tidy..."
        go mod tidy
    fi
    cd ..
else
    echo -e "   ${RED}❌${NC} src/go.mod missing"
    ERRORS=$((ERRORS + 1))
fi

# Check AWS CLI (optional for local dev)
echo ""
echo "5. Checking AWS CLI (optional for AWS deployment)..."
if command -v aws > /dev/null 2>&1; then
    echo -e "   ${GREEN}✅${NC} AWS CLI is installed"
    
    if aws sts get-caller-identity > /dev/null 2>&1; then
        echo -e "   ${GREEN}✅${NC} AWS credentials are configured"
        AWS_ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
        echo "      Account: $AWS_ACCOUNT"
    else
        echo -e "   ${YELLOW}⚠️${NC}  AWS credentials not configured (needed for AWS deployment)"
        echo "      Run: aws configure"
    fi
else
    echo -e "   ${YELLOW}⚠️${NC}  AWS CLI not installed (needed for AWS deployment)"
    echo "      Install: https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html"
fi

# Check Terraform (optional for local dev)
echo ""
echo "6. Checking Terraform (optional for AWS deployment)..."
if command -v terraform > /dev/null 2>&1; then
    TERRAFORM_VERSION=$(terraform --version | head -n1)
    echo -e "   ${GREEN}✅${NC} Terraform is installed ($TERRAFORM_VERSION)"
else
    echo -e "   ${YELLOW}⚠️${NC}  Terraform not installed (needed for AWS deployment)"
    echo "      Install: https://learn.hashicorp.com/tutorials/terraform/install-cli"
fi

# Check Docker Compose version
echo ""
echo "7. Checking Docker Compose..."
if docker compose version > /dev/null 2>&1; then
    COMPOSE_VERSION=$(docker compose version)
    echo -e "   ${GREEN}✅${NC} Docker Compose is available ($COMPOSE_VERSION)"
else
    echo -e "   ${RED}❌${NC} Docker Compose not available"
    ERRORS=$((ERRORS + 1))
fi

# Summary
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✅ All checks passed!${NC}"
    echo ""
    echo "You're ready to deploy!"
    echo ""
    echo "Next steps:"
    echo "  1. For local development:"
    echo "     ./run-local.sh"
    echo ""
    echo "  2. For AWS deployment (requires AWS CLI + Terraform):"
    echo "     ./deploy.sh"
    echo ""
    echo "  3. Read the documentation:"
    echo "     cat README.md"
    echo "     cat DEPLOYMENT_GUIDE.md"
else
    echo -e "${RED}❌ Found $ERRORS issue(s)${NC}"
    echo ""
    echo "Please fix the issues above and run this script again."
fi
echo "=========================================="
