#!/bin/bash
# setup-local-db.sh - Initialize MySQL database locally for development

set -e

echo "=========================================="
echo "🗄️  Setting up Local MySQL Database"
echo "=========================================="

# Check if MySQL container is running
if ! docker ps | grep -q shopping-cart-mysql; then
    echo "❌ MySQL container is not running"
    echo "Please run: docker-compose up -d mysql"
    exit 1
fi

echo "✅ MySQL container is running"

# Wait for MySQL to be ready
echo "⏳ Waiting for MySQL to be ready..."
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if docker exec shopping-cart-mysql mysqladmin ping -h localhost -u admin -pMySecurePass123! --silent 2>/dev/null; then
        echo "✅ MySQL is ready!"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    echo -n "."
    sleep 2
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo ""
    echo "❌ MySQL failed to start"
    exit 1
fi

# Run the setup SQL script
echo ""
echo "📝 Running database setup script..."
docker exec -i shopping-cart-mysql mysql -u admin -pMySecurePass123! shopping_cart_db < src/db/setup.sql

echo ""
echo "✅ Database setup complete!"
echo ""
echo "To connect to MySQL:"
echo "  docker exec -it shopping-cart-mysql mysql -u admin -pMySecurePass123! shopping_cart_db"
echo ""
