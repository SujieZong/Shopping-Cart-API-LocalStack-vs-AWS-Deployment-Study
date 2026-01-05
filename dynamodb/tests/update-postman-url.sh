#!/bin/bash
#
# Script to automatically update Postman collection with ALB URL from Terraform
# Usage: ./update-postman-url.sh
#

set -e

TERRAFORM_DIR="../terraform"
POSTMAN_COLLECTION="Shopping_Cart_Consistency_Test.postman_collection.json"

echo "==============================================="
echo "Updating Postman Collection with ALB URL"
echo "==============================================="

# Check if Terraform directory exists
if [ ! -d "$TERRAFORM_DIR" ]; then
    echo "ERROR: Terraform directory not found at $TERRAFORM_DIR"
    exit 1
fi

# Check if Postman collection exists
if [ ! -f "$POSTMAN_COLLECTION" ]; then
    echo "ERROR: Postman collection not found: $POSTMAN_COLLECTION"
    exit 1
fi

# Get ALB URL from Terraform output
echo "Fetching ALB URL from Terraform..."
cd "$TERRAFORM_DIR"

# Check if terraform state exists
if [ ! -f "terraform.tfstate" ]; then
    echo "ERROR: terraform.tfstate not found. Please run 'terraform apply' first."
    exit 1
fi

# Extract ALB URL
ALB_URL=$(terraform output -raw alb_url 2>/dev/null || echo "")

if [ -z "$ALB_URL" ]; then
    echo "ERROR: Could not retrieve ALB URL from Terraform output"
    echo "Make sure Terraform has been applied successfully"
    exit 1
fi

echo "ALB URL found: $ALB_URL"

# Go back to script directory
cd - > /dev/null

# Update Postman collection using jq (if available) or sed
if command -v jq &> /dev/null; then
    echo "Updating Postman collection using jq..."
    
    # Create temporary file with updated URL
    jq --arg url "$ALB_URL" '.variable[0].value = $url' "$POSTMAN_COLLECTION" > "${POSTMAN_COLLECTION}.tmp"
    
    # Replace original file
    mv "${POSTMAN_COLLECTION}.tmp" "$POSTMAN_COLLECTION"
    
    echo "✅ Successfully updated $POSTMAN_COLLECTION"
    echo "   base_url is now: $ALB_URL"
else
    echo "jq not found, using sed..."
    
    # Use sed to replace the base_url value
    # This is more fragile but works without jq
    sed -i.bak 's|"value": "http://[^"]*"|"value": "'"$ALB_URL"'"|' "$POSTMAN_COLLECTION"
    
    echo "✅ Successfully updated $POSTMAN_COLLECTION"
    echo "   base_url is now: $ALB_URL"
    echo "   (Backup saved as ${POSTMAN_COLLECTION}.bak)"
fi

echo ""
echo "You can now import this collection into Postman:"
echo "  $POSTMAN_COLLECTION"
echo ""
echo "==============================================="
