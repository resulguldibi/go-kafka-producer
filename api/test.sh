#!/bin/bash

# Test script for Go Kafka Producer

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"

echo -e "${YELLOW}Testing Go Kafka Producer API...${NC}\n"

# Test 1: Health check
echo -e "${YELLOW}1. Testing health endpoint...${NC}"
response=$(curl -s -w "\n%{http_code}" "$API_URL/protected/health")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n1)

if [ "$http_code" -eq 200 ]; then
    echo -e "${GREEN}✓ Health check passed${NC}"
    echo "Response: $body"
else
    echo -e "${RED}✗ Health check failed (HTTP $http_code)${NC}"
    echo "Response: $body"
fi

echo ""

# Test 2: Send valid events
echo -e "${YELLOW}2. Testing valid events...${NC}"
valid_json='[{
    "eventtimestamp": 1746788536758340000,
    "eventtime": "2025-05-09T14:02:16.75834+03:00",
    "id": "34B2D783-D297-D6B6-E063-4918060A0F70",
    "domain": "ForeignTrade",
    "subdomain": "Exchange",
    "code": "MoneyTransferOutgoingSwiftSent",
    "version": "1.0",
    "branchid": 8000,
    "channelid": 37,
    "customerid": 100537117,
    "userid": 78942,
    "payload": "EhAyMjA5R0tHRDI1MDAwNjY5GKERIMA+KgwIuMH3wAYQsLr85AIwnab4LzojR0VOTcSwVCDEsE7FnkFBVCBBTk9OxLBNIMWexLBSS0VUxLBKVllFTsSwIEtVUlVMQUNBSyBHRU5NxLBUIMSwTsWeQUFUIEEgxLBMS0JBSEFSIE1BSC4gNjA2IENELiBOTzogMTggMDY1NTAgw4dBTktBWUEgQU5LQVJBWgJUUnnsUbgehW26QIkB7FG4HoVtukCSAQNFVVKiAQE48QFJLv8h/SZGQIgClyeyAgNFVVKIA5AnkQNxrIvbaJBFQJkDSS7/If0mRkChA3Gsi9tokEVAqQNhHFw65vzxP+IDHkdBUkFOVEkgRklOQU5TQUwgS0lSQUxBTUEgQS5TLuoDAypUUvoDAlRSggQLVEdCQVRSSVNYWFiKBAtUR0JBVFJJU1hYWJIET1R1cmvEsXllIEdhcmFudMSxIEJhbmthc8SxIEEuIHMuO0xFVkVOVCBOSVNQRVRJWUUgTUFILiBBWVRBUkNBRC4gTk8uMiBCRVNJS1RBUyCaBAtUR0JBVFJJU1hYWMoEGQoGRjAwNjkzEKERGgwIuMH3wAYQsLr85AIRCBSI5MDAzMzcyIE5PTFUgRklOUyBLSVJMIFNPWkxTIFRBS1NUigUHQkVERUzEsLoGCERUSVRHREhPwgYBQeAHkCfyBwNFVVL6BwFIkggaVFI1MDAwMDYyMDAwMzgxMDAwMDkwNDUyMDOaCAtUR0JBVFJJU1hYWKIIAUGwCCW6CAc0Myw2ODQz2ggIOTk5MDAyMTE="
}]'

response=$(curl -s -w "\n%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  -d "$valid_json" \
  "$API_URL/events")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n1)

if [ "$http_code" -eq 200 ]; then
    echo -e "${GREEN}✓ Valid events test passed${NC}"
    echo "Response: $body"
else
    echo -e "${RED}✗ Valid events test failed (HTTP $http_code)${NC}"
    echo "Response: $body"
fi

echo ""

# Test 3: Send invalid events (missing required fields)
echo -e "${YELLOW}3. Testing invalid events...${NC}"
invalid_json='[{
    "eventtimestamp": 1746788536758340000,
    "eventtime": "2025-05-09T14:02:16.75834+03:00",
    "id": "invalid-event-id",
    "domain": "",
    "subdomain": "Exchange",
    "code": "",
    "version": "1.0",
    "branchid": 8000,
    "channelid": 37,
    "customerid": 100537117,
    "userid": 78942,
    "payload": "test"
}]'

response=$(curl -s -w "\n%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  -d "$invalid_json" \
  "$API_URL/events")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n1)

if [ "$http_code" -eq 200 ]; then
    echo -e "${GREEN}✓ Invalid events test passed${NC}"
    echo "Response: $body"
    
    # Check if the invalid event ID is in invalidEventIds
    if echo "$body" | grep -q "invalid-event-id" && echo "$body" | grep -q "invalidEventIds"; then
        echo -e "${GREEN}✓ Invalid event correctly identified${NC}"
    else
        echo -e "${RED}✗ Invalid event not correctly identified${NC}"
    fi
else
    echo -e "${RED}✗ Invalid events test failed (HTTP $http_code)${NC}"
    echo "Response: $body"
fi

echo ""

# Test 4: Send mixed events (valid and invalid)
echo -e "${YELLOW}4. Testing mixed events...${NC}"
mixed_json='[{
    "eventtimestamp": 1746788536758340000,
    "eventtime": "2025-05-09T14:02:16.75834+03:00",
    "id": "valid-event-1",
    "domain": "TestDomain",
    "subdomain": "TestSubdomain",
    "code": "TestCode",
    "version": "1.0",
    "branchid": 8000,
    "channelid": 37,
    "customerid": 100537117,
    "userid": 78942,
    "payload": "test"
},{
    "eventtimestamp": 1746788536758340000,
    "eventtime": "2025-05-09T14:02:16.75834+03:00",
    "id": "invalid-event-2",
    "domain": "",
    "subdomain": "",
    "code": "",
    "version": "1.0",
    "branchid": 8000,
    "channelid": 37,
    "customerid": 100537117,
    "userid": 78942,
    "payload": "test"
}]'

response=$(curl -s -w "\n%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  -d "$mixed_json" \
  "$API_URL/events")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n1)

if [ "$http_code" -eq 200 ]; then
    echo -e "${GREEN}✓ Mixed events test passed${NC}"
    echo "Response: $body"
else
    echo -e "${RED}✗ Mixed events test failed (HTTP $http_code)${NC}"
    echo "Response: $body"
fi

echo -e "\n${YELLOW}Testing completed!${NC}"
