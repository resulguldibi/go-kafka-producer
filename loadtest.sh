#!/bin/bash

# Load Test Script for Go Kafka Producer API

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"

echo -e "${BLUE}Go Kafka Producer - Load Test Tool${NC}\n"

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -d, --duration SECONDS    Test duration in seconds (default: 30)"
    echo "  -g, --goroutines NUMBER   Number of concurrent goroutines (default: 10)"
    echo "  -e, --events NUMBER       Number of events per request (default: 1)"
    echo "  -D, --delay MILLISECONDS  Delay between requests in ms (default: 100)"
    echo "  -u, --url URL             API base URL (default: http://localhost:8080)"
    echo "  -v, --verbose             Enable verbose output"
    echo "  -h, --help                Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Default test (30s, 10 goroutines)"
    echo "  $0 -d 60 -g 20                       # 60 seconds with 20 goroutines"
    echo "  $0 -d 120 -g 50 -e 5 -D 50          # 2 minutes, 50 goroutines, 5 events/req, 50ms delay"
    echo "  $0 -d 10 -g 5 -v                     # Quick test with verbose output"
}

# Default values
DURATION=30
GOROUTINES=10
EVENTS=1
DELAY=100
VERBOSE=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--duration)
            DURATION="$2"
            shift 2
            ;;
        -g|--goroutines)
            GOROUTINES="$2"
            shift 2
            ;;
        -e|--events)
            EVENTS="$2"
            shift 2
            ;;
        -D|--delay)
            DELAY="$2"
            shift 2
            ;;
        -u|--url)
            API_URL="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="-verbose"
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            show_usage
            exit 1
            ;;
    esac
done

# Validate inputs
if ! [[ "$DURATION" =~ ^[0-9]+$ ]] || [ "$DURATION" -lt 1 ]; then
    echo -e "${RED}Error: Duration must be a positive integer${NC}"
    exit 1
fi

if ! [[ "$GOROUTINES" =~ ^[0-9]+$ ]] || [ "$GOROUTINES" -lt 1 ]; then
    echo -e "${RED}Error: Goroutines must be a positive integer${NC}"
    exit 1
fi

if ! [[ "$EVENTS" =~ ^[0-9]+$ ]] || [ "$EVENTS" -lt 1 ]; then
    echo -e "${RED}Error: Events per request must be a positive integer${NC}"
    exit 1
fi

if ! [[ "$DELAY" =~ ^[0-9]+$ ]]; then
    echo -e "${RED}Error: Delay must be a non-negative integer${NC}"
    exit 1
fi

# Check if API is running
echo -e "${YELLOW}Checking if API is running...${NC}"
if ! curl -s -f "$API_URL/protected/health" > /dev/null; then
    echo -e "${RED}Error: API is not running at $API_URL${NC}"
    echo "Please make sure the Go Kafka Producer API is running with:"
    echo "  docker-compose up -d"
    exit 1
fi
echo -e "${GREEN}✓ API is running${NC}"

# Build the load test application
echo -e "${YELLOW}Building load test application...${NC}"
cd "$(dirname "$0")/loadtest"
if ! go build -o loadtest .; then
    echo -e "${RED}Error: Failed to build load test application${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Load test application built${NC}"

# Run the load test
echo -e "${YELLOW}Starting load test...${NC}"
echo "Configuration:"
echo "  Duration: ${DURATION} seconds"
echo "  Goroutines: ${GOROUTINES}"
echo "  Events per request: ${EVENTS}"
echo "  Delay between requests: ${DELAY} ms"
echo "  API URL: ${API_URL}"
echo "  Verbose: $([ -n "$VERBOSE" ] && echo "Yes" || echo "No")"
echo ""

./loadtest \
    -duration="$DURATION" \
    -goroutines="$GOROUTINES" \
    -events="$EVENTS" \
    -delay="$DELAY" \
    -url="$API_URL" \
    $VERBOSE

echo -e "\n${GREEN}Load test completed!${NC}"

# Cleanup
rm -f loadtest
