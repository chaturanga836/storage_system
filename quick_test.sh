#!/bin/bash
# Quick Testing Script for EC2 Deployment

echo "🧪 QUICK TESTING SCRIPT FOR STORAGE SYSTEM"
echo "=========================================="

# Set EC2 IP (replace with your actual EC2 IP)
EC2_IP="13.232.166.185"  # Update this with your EC2 IP
if [ "$1" != "" ]; then
    EC2_IP=$1
fi

echo "🎯 Testing services on: $EC2_IP"
echo ""

# Function to test an endpoint
test_endpoint() {
    local service=$1
    local port=$2
    local url="http://$EC2_IP:$port/health"
    
    echo -n "Testing $service (port $port)... "
    
    response=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 "$url" 2>/dev/null)
    
    if [ "$response" = "200" ]; then
        echo "✅ HEALTHY"
        return 0
    else
        echo "❌ UNHEALTHY (HTTP $response)"
        return 1
    fi
}

# Test all services
echo "🏥 HEALTH CHECK RESULTS:"
echo "------------------------"

total=0
passed=0

services=(
    "auth-gateway:8080"
    "tenant-node:8001"
    "operation-node:8086"
    "cbo-engine:8088"  
    "metadata-catalog:8087"
    "monitoring:8089"
    "grafana:3000"
    "prometheus:9090"
)

for service_port in "${services[@]}"; do
    IFS=':' read -r service port <<< "$service_port"
    total=$((total + 1))
    
    if test_endpoint "$service" "$port"; then
        passed=$((passed + 1))
    fi
done

echo ""
echo "📊 SUMMARY:"
echo "----------"
echo "Total Services: $total"
echo "Healthy: $passed"
echo "Unhealthy: $((total - passed))"

success_rate=$((passed * 100 / total))
echo "Success Rate: $success_rate%"

if [ $passed -eq $total ]; then
    echo ""
    echo "🎉 ALL SERVICES ARE HEALTHY! 🎉"
    echo "Your storage system is fully operational!"
else
    echo ""
    echo "⚠️  Some services are unhealthy. Check logs:"
    echo "   docker-compose logs"
fi

echo ""
echo "📋 NEXT STEPS:"
echo "1. If all healthy: Run comprehensive tests with 'python test_runner.py $EC2_IP'"
echo "2. If any unhealthy: Check service logs and restart if needed"
echo "3. Monitor with: docker-compose ps"
