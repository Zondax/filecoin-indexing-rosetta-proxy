#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
MOCKERY_VERSION="latest"
MOCKS_DIR="tests/mocks"
BOILERPLATE_FILE="tests/mock-boilerplate.txt"

# Ensure mockery is installed with correct version
install_mockery() {
    go install github.com/vektra/mockery/v3@${MOCKERY_VERSION}
}

# Generate mocks for internal interfaces
generate_internal_mocks() {
    echo -e "${GREEN}Generating mocks...${NC}"
    
    mockery --config .mockery.yaml
}

# Main execution
main() {
    install_mockery
    generate_internal_mocks
}

main