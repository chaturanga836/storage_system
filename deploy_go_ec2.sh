#!/bin/bash
# Quick Go Control Plane Deployment Script for EC2

echo "🚀 GO CONTROL PLANE DEPLOYMENT SCRIPT"
echo "====================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check if we're on the right system
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    print_error "This script is designed for Linux (EC2). Please run on your EC2 instance."
    exit 1
fi

print_header "1. Checking Go installation..."
if ! command -v go &> /dev/null; then
    print_warning "Go not found. Installing Go 1.21..."
    
    # Install Go
    cd /tmp
    wget -q https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    
    if [ -f "go1.21.6.linux-amd64.tar.gz" ]; then
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
        
        # Add to PATH
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo 'export GOPATH=$HOME/go' >> ~/.bashrc
        echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
        
        export PATH=$PATH:/usr/local/go/bin
        export GOPATH=$HOME/go
        export PATH=$PATH:$GOPATH/bin
        
        print_status "Go installed successfully"
        go version
    else
        print_error "Failed to download Go. Please install manually."
        exit 1
    fi
else
    print_status "Go is already installed: $(go version)"
fi

print_header "2. Setting up project directory..."
cd $HOME

# Check if directory exists
if [ ! -d "storage_control_plane" ]; then
    print_error "storage_control_plane directory not found. Please clone the repository first:"
    echo "  git clone <your-repo-url>"
    echo "  cd storage_control_plane"
    exit 1
fi

cd storage_control_plane
print_status "Working in: $(pwd)"

print_header "3. Installing dependencies..."
if [ -f "go.mod" ]; then
    go mod download
    print_status "Dependencies downloaded"
else
    print_warning "go.mod not found. Initializing module..."
    go mod init storage_control_plane
    
    # Add common dependencies
    go get github.com/joho/godotenv
    go get github.com/gorilla/mux
    go get github.com/rs/cors
    print_status "Module initialized with basic dependencies"
fi

print_header "4. Setting up environment..."
if [ ! -f ".env" ]; then
    if [ -f ".env.example" ]; then
        cp .env.example .env
        print_status "Environment file created from template"
    else
        # Create basic .env file
        cat > .env << EOF
# Go Control Plane Configuration
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info

# Database Configuration (optional)
DB_HOST=localhost
DB_PORT=5432
DB_NAME=storage_control
DB_USER=postgres
DB_PASSWORD=password

# Redis Configuration (optional)
REDIS_HOST=localhost
REDIS_PORT=6379

# Python Services (update with your Python EC2 IP)
PYTHON_SERVICES_HOST=localhost
PYTHON_AUTH_GATEWAY=http://localhost:8080
PYTHON_TENANT_NODE=http://localhost:8001
PYTHON_METADATA_CATALOG=http://localhost:8087
EOF
        print_status "Basic environment file created"
    fi
else
    print_status "Environment file already exists"
fi

print_header "5. Building Go application..."
if go build -o storage-control-plane .; then
    print_status "Build successful: storage-control-plane binary created"
    ls -la storage-control-plane
else
    print_error "Build failed. Please check the error messages above."
    exit 1
fi

print_header "6. Setting up systemd service..."
sudo tee /etc/systemd/system/storage-control-plane.service > /dev/null <<EOF
[Unit]
Description=Storage Control Plane
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$HOME/storage_control_plane
ExecStart=$HOME/storage_control_plane/storage-control-plane
Restart=always
RestartSec=10
Environment=PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
EnvironmentFile=$HOME/storage_control_plane/.env

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable storage-control-plane
print_status "Systemd service configured"

print_header "7. Starting service..."
sudo systemctl start storage-control-plane

# Wait a moment for service to start
sleep 3

if sudo systemctl is-active --quiet storage-control-plane; then
    print_status "Service started successfully"
else
    print_error "Service failed to start. Checking logs..."
    sudo journalctl -u storage-control-plane --no-pager -l
    exit 1
fi

print_header "8. Running health checks..."
sleep 2

# Function to check endpoint
check_endpoint() {
    local url=$1
    local name=$2
    
    if curl -s --connect-timeout 5 "$url" > /dev/null; then
        print_status "✅ $name is healthy"
        return 0
    else
        print_warning "❌ $name is not responding"
        return 1
    fi
}

# Check all endpoints
healthy_count=0
total_endpoints=7

endpoints=(
    "http://localhost:8080/health:Main Service"
    "http://localhost:8000/health:Tenant Node"
    "http://localhost:8081/health:Operation Node"
    "http://localhost:8082/health:CBO Engine"
    "http://localhost:8083/health:Metadata Catalog"
    "http://localhost:8084/health:Monitoring"
    "http://localhost:8085/health:Query Interpreter"
)

for endpoint in "${endpoints[@]}"; do
    IFS=':' read -r url name <<< "$endpoint"
    if check_endpoint "$url" "$name"; then
        ((healthy_count++))
    fi
done

print_header "9. Deployment Summary"
echo "======================================"
echo "Service Status: $(sudo systemctl is-active storage-control-plane)"
echo "Service Enabled: $(sudo systemctl is-enabled storage-control-plane)"
echo "Healthy Endpoints: $healthy_count/$total_endpoints"
echo "Process ID: $(pgrep storage-control-plane || echo 'Not running')"
echo "Memory Usage: $(ps -o pid,vsz,rss,comm -p $(pgrep storage-control-plane) 2>/dev/null | tail -1 || echo 'N/A')"

# Get public IP
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 2>/dev/null || echo "unknown")
PRIVATE_IP=$(curl -s http://169.254.169.254/latest/meta-data/local-ipv4 2>/dev/null || echo "unknown")

echo ""
echo "🌐 Access URLs:"
echo "  Public:  http://$PUBLIC_IP:8080"
echo "  Private: http://$PRIVATE_IP:8080"
echo "  Local:   http://localhost:8080"

if [ $healthy_count -eq $total_endpoints ]; then
    echo ""
    echo "🎉 DEPLOYMENT SUCCESSFUL! 🎉"
    echo "Your Go Control Plane is fully operational!"
    echo ""
    echo "📋 Next Steps:"
    echo "1. Update security groups to allow access on port 8080"
    echo "2. Test endpoints: curl http://localhost:8080/health"
    echo "3. Monitor logs: sudo journalctl -u storage-control-plane -f"
    echo "4. Configure connection to Python services in .env file"
else
    echo ""
    echo "⚠️  PARTIAL DEPLOYMENT"
    echo "Some endpoints are not responding. Check logs:"
    echo "  sudo journalctl -u storage-control-plane -f"
fi

echo ""
echo "📊 Service Management Commands:"
echo "  Start:   sudo systemctl start storage-control-plane"
echo "  Stop:    sudo systemctl stop storage-control-plane"
echo "  Restart: sudo systemctl restart storage-control-plane"
echo "  Status:  sudo systemctl status storage-control-plane"
echo "  Logs:    sudo journalctl -u storage-control-plane -f"
