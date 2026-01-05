#!/bin/bash
# cleanup.sh - Force cleanup all AWS resources

set -e

echo "=========================================="
echo "[CLEANUP] FORCE CLEANUP - HW8 MySQL Project"
echo "=========================================="

AWS_REGION="us-west-2"
PROJECT_NAME="hw8-mysql"

# 1. Force delete ECR repository
echo -e "\n[STEP 1] Force deleting ECR repository..."
aws ecr delete-repository \
    --repository-name ${PROJECT_NAME}-app \
    --region ${AWS_REGION} \
    --force 2>/dev/null || echo "ECR repository already deleted or doesn't exist"

# 2. Remove ECR from Terraform state
echo -e "\n[STEP 2] Removing ECR from Terraform state..."
cd terraform
terraform state rm aws_ecr_repository.app 2>/dev/null || echo "ECR not in state"

# 3. Run terraform destroy
echo -e "\n[STEP 3] Running terraform destroy..."
terraform destroy -auto-approve

# 4. Clean up Terraform files
echo -e "\n[STEP 4] Cleaning up Terraform files..."
rm -rf .terraform .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup

cd ..

echo ""
echo "=========================================="
echo "[SUCCESS] FORCE CLEANUP COMPLETE!"
echo "=========================================="
echo ""
echo "All resources have been forcefully removed."
echo "You can now run ./deploy.sh for a fresh deployment."