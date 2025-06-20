#!/bin/bash

# Pre-commit script that runs all CI checks locally
# This matches the GitHub Actions CI pipeline exactly

set -e

echo "🚀 Running pre-commit CI checks..."
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track failures
FAILED=0

# Function to run a command and track failures
run_check() {
    local name="$1"
    local cmd="$2"
    
    echo -e "\n${BLUE}🔄 Running $name...${NC}"
    if eval "$cmd"; then
        echo -e "${GREEN}✅ $name passed${NC}"
    else
        echo -e "${RED}❌ $name failed${NC}"
        FAILED=1
    fi
}

# Make sure we're in the project root
if [ ! -f "go.mod" ]; then
    echo -e "${RED}❌ Please run this script from the project root directory${NC}"
    exit 1
fi

# Check if required tools are installed
echo -e "${BLUE}🔍 Checking required tools...${NC}"

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
fi

if ! command -v make >/dev/null 2>&1; then
    echo -e "${RED}❌ Make is not installed${NC}"
    exit 1
fi

# Optional tools
if ! command -v gosec >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest${NC}"
fi

if ! command -v helm >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  helm not installed. Some validation checks will be skipped${NC}"
fi

if ! command -v kubectl >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  kubectl not installed. Some validation checks will be skipped${NC}"
fi

echo -e "${GREEN}✅ Tool check complete${NC}"

# Run CI checks (matching the GitHub Actions pipeline)
run_check "Unit Tests" "make ci-test"
run_check "Linting" "make ci-lint" 
run_check "Security Scan" "make ci-security"
run_check "Manifest Validation" "make ci-validate"

echo -e "\n================================================"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All CI checks passed! Your code is ready for PR.${NC}"
    echo -e "${GREEN}You can safely push your changes.${NC}"
    exit 0
else
    echo -e "${RED}❌ Some CI checks failed. Please fix the issues above.${NC}"
    echo -e "${RED}Your PR will likely fail CI until these are resolved.${NC}"
    exit 1
fi 