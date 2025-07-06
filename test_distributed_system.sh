#!/bin/bash
# Combined Testing Script for Both Python and Go Systems

echo "🌟 DISTRIBUTED STORAGE SYSTEM - COMPREHENSIVE TESTING"
echo "====================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}[PHASE]${NC} $1"
}

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get IPs from command line or use defaults
PYTHON_IP=${1:-"localhost"}
GO_IP=${2:-"localhost"}

if [ "$PYTHON_IP" = "localhost" ] && [ "$GO_IP" = "localhost" ]; then
    print_warning "Using localhost for both services. Provide IPs for remote testing:"
    echo "  Usage: $0 <python-ec2-ip> <go-ec2-ip>"
    echo ""
fi

echo "🎯 Python Microservices: $PYTHON_IP"
echo "🎯 Go Control Plane: $GO_IP"
echo ""

# Check if Python is available for Python scripts
if ! command -v python3 &> /dev/null; then
    print_error "Python3 not found. Please install Python3 to run tests."
    exit 1
fi

# Install required Python packages if needed
if ! python3 -c "import requests" 2>/dev/null; then
    print_status "Installing Python requests library..."
    pip3 install requests
fi

# Phase 1: Quick Health Checks
print_header "PHASE 1: QUICK HEALTH CHECKS"
echo "----------------------------------------"

# Function to test an endpoint
test_endpoint() {
    local service=$1
    local ip=$2
    local port=$3
    local url="http://$ip:$port/health"
    
    echo -n "Testing $service ($ip:$port)... "
    
    response=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$url" 2>/dev/null)
    
    if [ "$response" = "200" ]; then
        echo "✅ HEALTHY"
        return 0
    else
        echo "❌ UNHEALTHY (HTTP $response)"
        return 1
    fi
}

# Test Python services
echo "🐍 Python Microservices:"
python_total=0
python_healthy=0

python_services=(
    "auth-gateway:$PYTHON_IP:8080"
    "tenant-node:$PYTHON_IP:8001"
    "operation-node:$PYTHON_IP:8086"
    "cbo-engine:$PYTHON_IP:8088"
    "metadata-catalog:$PYTHON_IP:8087"
    "monitoring:$PYTHON_IP:8089"
)

for service_info in "${python_services[@]}"; do
    IFS=':' read -r service ip port <<< "$service_info"
    python_total=$((python_total + 1))
    
    if test_endpoint "$service" "$ip" "$port"; then
        python_healthy=$((python_healthy + 1))
    fi
done

echo ""
echo "🚀 Go Control Plane:"
go_total=0
go_healthy=0

go_services=(
    "main-service:$GO_IP:8080"
    "tenant-node:$GO_IP:8000"
    "operation-node:$GO_IP:8081"
    "cbo-engine:$GO_IP:8082"
    "metadata-catalog:$GO_IP:8083"
    "monitoring:$GO_IP:8084"
    "query-interpreter:$GO_IP:8085"
)

for service_info in "${go_services[@]}"; do
    IFS=':' read -r service ip port <<< "$service_info"
    go_total=$((go_total + 1))
    
    if test_endpoint "$service" "$ip" "$port"; then
        go_healthy=$((go_healthy + 1))
    fi
done

echo ""
echo "📊 Health Check Summary:"
echo "  Python Services: $python_healthy/$python_total healthy"
echo "  Go Services: $go_healthy/$go_total healthy"

# Phase 2: Python System Testing
print_header "PHASE 2: PYTHON MICROSERVICES TESTING"
echo "----------------------------------------"

if [ $python_healthy -gt 0 ]; then
    print_status "Running Python integration tests..."
    if python3 test_runner.py "$PYTHON_IP" 2>/dev/null; then
        python_tests_passed=true
        print_status "✅ Python tests completed successfully"
    else
        python_tests_passed=false
        print_warning "⚠️  Python tests had some issues"
    fi
else
    print_error "❌ No Python services healthy - skipping Python tests"
    python_tests_passed=false
fi

# Phase 3: Go System Testing
print_header "PHASE 3: GO CONTROL PLANE TESTING"
echo "----------------------------------------"

if [ $go_healthy -gt 0 ]; then
    print_status "Running Go control plane tests..."
    if python3 test_go_control_plane.py "$GO_IP" "$PYTHON_IP" 2>/dev/null; then
        go_tests_passed=true
        print_status "✅ Go tests completed successfully"
    else
        go_tests_passed=false
        print_warning "⚠️  Go tests had some issues"
    fi
else
    print_error "❌ No Go services healthy - skipping Go tests"
    go_tests_passed=false
fi

# Phase 4: Cross-System Integration
print_header "PHASE 4: CROSS-SYSTEM INTEGRATION"
echo "----------------------------------------"

if [ $python_healthy -gt 0 ] && [ $go_healthy -gt 0 ]; then
    print_status "Testing cross-system communication..."
    
    # Test if Go can reach Python
    echo -n "Go → Python connectivity... "
    if curl -s --connect-timeout 3 "http://$PYTHON_IP:8080/health" > /dev/null; then
        echo "✅ OK"
        cross_system_ok=true
    else
        echo "❌ FAILED"
        cross_system_ok=false
    fi
    
    # Test if Python can reach Go (if different IPs)
    if [ "$PYTHON_IP" != "$GO_IP" ]; then
        echo -n "Python → Go connectivity... "
        if curl -s --connect-timeout 3 "http://$GO_IP:8080/health" > /dev/null; then
            echo "✅ OK"
        else
            echo "❌ FAILED"
            cross_system_ok=false
        fi
    fi
    
else
    print_warning "Skipping cross-system testing (not all systems healthy)"
    cross_system_ok=false
fi

# Phase 5: Load Testing
print_header "PHASE 5: BASIC LOAD TESTING"
echo "----------------------------------------"

if [ $python_healthy -gt 0 ] || [ $go_healthy -gt 0 ]; then
    print_status "Running basic load tests..."
    
    # Test Python system load
    if [ $python_healthy -gt 0 ]; then
        echo -n "Python system load test... "
        # Simple load test with curl
        success_count=0
        for i in {1..10}; do
            if curl -s --connect-timeout 2 "http://$PYTHON_IP:8001/health" > /dev/null; then
                success_count=$((success_count + 1))
            fi
        done
        
        if [ $success_count -ge 8 ]; then
            echo "✅ PASSED (8/10 requests successful)"
            python_load_ok=true
        else
            echo "❌ FAILED ($success_count/10 requests successful)"
            python_load_ok=false
        fi
    fi
    
    # Test Go system load
    if [ $go_healthy -gt 0 ]; then
        echo -n "Go system load test... "
        success_count=0
        for i in {1..10}; do
            if curl -s --connect-timeout 2 "http://$GO_IP:8080/health" > /dev/null; then
                success_count=$((success_count + 1))
            fi
        done
        
        if [ $success_count -ge 8 ]; then
            echo "✅ PASSED ($success_count/10 requests successful)"
            go_load_ok=true
        else
            echo "❌ FAILED ($success_count/10 requests successful)"
            go_load_ok=false
        fi
    fi
else
    print_warning "Skipping load testing (no systems healthy)"
fi

# Final Report
print_header "FINAL REPORT"
echo "============================================"

echo "🏥 SYSTEM HEALTH:"
echo "  Python Microservices: $python_healthy/$python_total services healthy"
echo "  Go Control Plane:     $go_healthy/$go_total services healthy"

echo ""
echo "🧪 TEST RESULTS:"
echo "  Python Integration:   ${python_tests_passed:-"N/A"}"
echo "  Go Integration:       ${go_tests_passed:-"N/A"}"
echo "  Cross-System Comm:    ${cross_system_ok:-"N/A"}"
echo "  Load Testing:         Python=${python_load_ok:-"N/A"}, Go=${go_load_ok:-"N/A"}"

echo ""
echo "🌐 SYSTEM ACCESS:"
echo "  Python Services:      http://$PYTHON_IP:8001 (tenant-node)"
echo "  Go Control Plane:     http://$GO_IP:8080 (main service)"

# Calculate overall score
total_score=0
max_score=0

# Health scores
if [ $python_total -gt 0 ]; then
    total_score=$((total_score + python_healthy))
    max_score=$((max_score + python_total))
fi

if [ $go_total -gt 0 ]; then
    total_score=$((total_score + go_healthy))
    max_score=$((max_score + go_total))
fi

# Test scores (each worth 2 points)
if [ "$python_tests_passed" = true ]; then total_score=$((total_score + 2)); fi
if [ "$go_tests_passed" = true ]; then total_score=$((total_score + 2)); fi
if [ "$cross_system_ok" = true ]; then total_score=$((total_score + 2)); fi
max_score=$((max_score + 6))

if [ "$python_load_ok" = true ]; then total_score=$((total_score + 1)); fi
if [ "$go_load_ok" = true ]; then total_score=$((total_score + 1)); fi
max_score=$((max_score + 2))

# Final assessment
if [ $max_score -gt 0 ]; then
    success_percentage=$((total_score * 100 / max_score))
else
    success_percentage=0
fi

echo ""
echo "📊 OVERALL ASSESSMENT:"
echo "  Score: $total_score/$max_score ($success_percentage%)"

if [ $success_percentage -ge 90 ]; then
    echo "  Status: 🎉 EXCELLENT - Both systems fully operational!"
elif [ $success_percentage -ge 75 ]; then
    echo "  Status: ✅ GOOD - Systems mostly operational"
elif [ $success_percentage -ge 50 ]; then
    echo "  Status: ⚠️  PARTIAL - Some issues need attention"
else
    echo "  Status: ❌ CRITICAL - Systems need immediate attention"
fi

echo ""
echo "📋 NEXT STEPS:"
if [ $success_percentage -ge 90 ]; then
    echo "  • Your distributed system is ready for production!"
    echo "  • Consider setting up monitoring and alerting"
    echo "  • Configure SSL/TLS for secure communication"
elif [ $success_percentage -ge 50 ]; then
    echo "  • Check logs for failing services: docker-compose logs"
    echo "  • Verify network connectivity between instances"
    echo "  • Restart any failing services"
else
    echo "  • Check service status: docker-compose ps"
    echo "  • Review deployment logs and error messages"
    echo "  • Verify EC2 security groups and network configuration"
fi

echo ""
echo "🔧 MONITORING COMMANDS:"
echo "  Python logs:  docker-compose logs -f"
echo "  Go logs:      sudo journalctl -u storage-control-plane -f"
echo "  System stats: htop, docker stats"

# Exit with appropriate code
if [ $success_percentage -ge 75 ]; then
    exit 0
else
    exit 1
fi
