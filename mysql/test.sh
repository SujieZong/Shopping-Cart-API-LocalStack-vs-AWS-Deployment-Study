#!/bin/bash

echo "=========================================="
echo "🧪 Running Performance Test"
echo "=========================================="

# 检查是本地还是AWS
if [ -z "$API_URL" ]; then
    echo "Testing local instance..."
    export API_URL="http://localhost:3000"
    
    # 检查本地服务是否运行
    if ! curl -s http://localhost:3000/health > /dev/null 2>&1; then
        echo "❌ Local service not running!"
        echo "Please start it first: cd src && go run main.go"
        exit 1
    fi
else
    echo "Testing remote instance: $API_URL"
fi

# 运行测试
cd tests
go run performance_test.go

# 检查结果
if [ -f "../mysql_test_results.json" ]; then
    echo "✅ Test completed!"
    mv ../mysql_test_results.json ../results/
    
    # 显示统计
    echo -e "\n📊 Test Statistics:"
    cat ../results/mysql_test_results.json | \
        jq '[.[] | .operation] | group_by(.) | map({(.[0]): length})'
    
    echo -e "\n📈 Average Response Times:"
    cat ../results/mysql_test_results.json | \
        jq 'group_by(.operation) | map({operation: .[0].operation, avg_ms: ([.[] | .response_time] | add / length | floor)})'
else
    echo "❌ Test failed - no results generated"
    exit 1
fi