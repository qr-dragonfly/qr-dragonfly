#!/bin/bash

# Deployment script for monorepo to multiple Heroku apps
# Usage: ./deploy.sh [service-name]
# Example: ./deploy.sh qr-service
# Or deploy all: ./deploy.sh all

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to deploy a service
deploy_service() {
    local service=$1
    local path=$2
    local remote=$3
    
    echo -e "${YELLOW}Deploying ${service}...${NC}"
    
    # Check if remote exists
    if ! git remote | grep -q "^${remote}$"; then
        echo -e "${RED}Error: Remote '${remote}' not found${NC}"
        echo "Add it with: git remote add ${remote} https://git.heroku.com/your-app-name.git"
        return 1
    fi
    
    # Deploy using git subtree
    git subtree push --prefix ${path} ${remote} main || {
        echo -e "${RED}Failed to deploy ${service}${NC}"
        return 1
    }
    
    echo -e "${GREEN}✓ ${service} deployed successfully${NC}"
}

# Main deployment logic
case "$1" in
    qr-service)
        deploy_service "QR Service" "backend/qr-service" "heroku-qr"
        ;;
    click-service)
        deploy_service "Click Service" "backend/click-service" "heroku-click"
        ;;
    user-service)
        deploy_service "User Service" "backend/user-service" "heroku-user"
        ;;
    frontend)
        deploy_service "Frontend" "frontend" "heroku-frontend"
        ;;
    all)
        echo -e "${YELLOW}Deploying all services...${NC}"
        deploy_service "QR Service" "backend/qr-service" "heroku-qr"
        deploy_service "Click Service" "backend/click-service" "heroku-click"
        deploy_service "User Service" "backend/user-service" "heroku-user"
        deploy_service "Frontend" "frontend" "heroku-frontend"
        echo -e "${GREEN}✓ All services deployed successfully${NC}"
        ;;
    *)
        echo "Usage: ./deploy.sh [service-name]"
        echo ""
        echo "Available services:"
        echo "  qr-service      - Deploy QR code generation service"
        echo "  click-service   - Deploy click tracking service"
        echo "  user-service    - Deploy user/auth service"
        echo "  frontend        - Deploy frontend application"
        echo "  all             - Deploy all services"
        echo ""
        echo "Example: ./deploy.sh qr-service"
        exit 1
        ;;
esac
