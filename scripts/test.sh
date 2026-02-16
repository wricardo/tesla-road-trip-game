#!/bin/bash

# Tesla Road Trip Game - Test Script
# Comprehensive testing with various options

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test options
VERBOSE=false
COVERAGE=false
BENCH=false
RACE=false
SHORT=false
PACKAGE=""

print_help() {
    echo "Tesla Road Trip Game - Test Script"
    echo ""
    echo "Usage: $0 [OPTIONS] [PACKAGE]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose     Run tests with verbose output"
    echo "  -c, --coverage    Generate coverage report"
    echo "  -b, --bench       Run benchmarks"
    echo "  -r, --race        Run tests with race detection"
    echo "  -s, --short       Run short tests only"
    echo "  --package PKG     Run tests for specific package"
    echo "  -h, --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                        # Run all tests"
    echo "  $0 -v -c                  # Verbose tests with coverage"
    echo "  $0 --package ./api        # Test only API package"
    echo "  $0 -r -s                  # Short tests with race detection"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -b|--bench)
            BENCH=true
            shift
            ;;
        -r|--race)
            RACE=true
            shift
            ;;
        -s|--short)
            SHORT=true
            shift
            ;;
        --package)
            PACKAGE="$2"
            shift 2
            ;;
        -h|--help)
            print_help
            exit 0
            ;;
        *)
            if [[ -z "$PACKAGE" && "$1" != -* ]]; then
                PACKAGE="$1"
                shift
            else
                echo -e "${RED}Unknown option: $1${NC}"
                print_help
                exit 1
            fi
            ;;
    esac
done

# Default package if none specified
if [[ -z "$PACKAGE" ]]; then
    PACKAGE="./..."
fi

echo -e "${BLUE}Tesla Road Trip Game - Test Runner${NC}"
echo -e "${BLUE}==================================${NC}"

# Build test command
TEST_CMD="go test"
TEST_FLAGS=""

if [[ "$VERBOSE" == true ]]; then
    TEST_FLAGS="$TEST_FLAGS -v"
fi

if [[ "$COVERAGE" == true ]]; then
    TEST_FLAGS="$TEST_FLAGS -coverprofile=coverage.out"
fi

if [[ "$RACE" == true ]]; then
    TEST_FLAGS="$TEST_FLAGS -race"
fi

if [[ "$SHORT" == true ]]; then
    TEST_FLAGS="$TEST_FLAGS -short"
fi

# Pre-test setup
echo -e "${YELLOW}Setting up tests...${NC}"

# Format code
echo -e "${BLUE}Formatting code...${NC}"
go fmt ./...

# Run vet
echo -e "${BLUE}Running go vet...${NC}"
if go vet ./...; then
    echo -e "${GREEN}✓ go vet passed${NC}"
else
    echo -e "${RED}✗ go vet failed${NC}"
    exit 1
fi

# Run tests
echo -e "${BLUE}Running tests...${NC}"
echo -e "${YELLOW}Command: $TEST_CMD $TEST_FLAGS $PACKAGE${NC}"

if $TEST_CMD $TEST_FLAGS $PACKAGE; then
    echo -e "${GREEN}✓ Tests passed${NC}"
    TEST_SUCCESS=true
else
    echo -e "${RED}✗ Tests failed${NC}"
    TEST_SUCCESS=false
fi

# Post-test actions
if [[ "$COVERAGE" == true && "$TEST_SUCCESS" == true ]]; then
    echo -e "${BLUE}Generating coverage report...${NC}"

    # Show coverage summary
    go tool cover -func=coverage.out | tail -1

    # Generate HTML report
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}✓ Coverage report saved to coverage.html${NC}"

    # Show coverage by package
    echo -e "${BLUE}Coverage by package:${NC}"
    go tool cover -func=coverage.out | grep -E "^[^/]*/" | sort -k3 -nr
fi

# Run benchmarks if requested
if [[ "$BENCH" == true && "$TEST_SUCCESS" == true ]]; then
    echo -e "${BLUE}Running benchmarks...${NC}"
    if go test -bench=. -benchmem $PACKAGE; then
        echo -e "${GREEN}✓ Benchmarks completed${NC}"
    else
        echo -e "${YELLOW}⚠ Benchmarks completed with issues${NC}"
    fi
fi

# Validate configurations
if [[ "$PACKAGE" == "./..." && "$TEST_SUCCESS" == true ]]; then
    echo -e "${BLUE}Validating game configurations...${NC}"
    if make validate >/dev/null 2>&1; then
        echo -e "${GREEN}✓ All configurations valid${NC}"
    else
        echo -e "${YELLOW}⚠ Configuration validation issues detected${NC}"
    fi
fi

# Final summary
echo -e "${BLUE}==================================${NC}"
if [[ "$TEST_SUCCESS" == true ]]; then
    echo -e "${GREEN}✓ All tests completed successfully${NC}"
    exit 0
else
    echo -e "${RED}✗ Test run failed${NC}"
    exit 1
fi