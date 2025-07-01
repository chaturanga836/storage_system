#!/bin/bash

# Multi-Tenant Storage System - Stop All Services

echo "🛑 Stopping Multi-Tenant Storage System Services"
echo "================================================="

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

if [ -f ".pids" ]; then
    echo -e "${YELLOW}Stopping services...${NC}"
    
    for pid in $(cat .pids); do
        if kill -0 $pid 2>/dev/null; then
            echo -e "${YELLOW}Stopping process $pid...${NC}"
            kill $pid
            sleep 1
            
            # Force kill if still running
            if kill -0 $pid 2>/dev/null; then
                echo -e "${RED}Force killing process $pid...${NC}"
                kill -9 $pid
            fi
        else
            echo -e "${GREEN}Process $pid already stopped${NC}"
        fi
    done
    
    rm -f .pids
    echo -e "${GREEN}All services stopped successfully!${NC}"
else
    echo -e "${YELLOW}No PID file found. Services may not be running.${NC}"
fi

# Clean up any remaining Python processes (optional)
echo -e "${YELLOW}Cleaning up any remaining Python processes...${NC}"
pkill -f "python main.py" 2>/dev/null || true

echo -e "${GREEN}Cleanup complete!${NC}"
