#!/bin/bash

# Test script for Redis Rate Limiter endpoints
# Make sure the server is running before executing this script

echo "Testing Redis Rate Limiter Endpoints"
echo "====================================="

# Test health endpoint
echo -e "\n1. Testing Health Endpoint:"
curl -s http://localhost:8080/health
echo ""

# Test fixed window endpoint multiple times
echo -e "\n2. Testing Fixed Window Rate Limiting (3 requests allowed per 10 seconds):"
for i in {1..5}; do
    echo "Request $i:"
    curl -s http://localhost:8080/fixed-window
    echo ""
    sleep 1
done

# Test with custom client ID
echo -e "\n3. Testing Fixed Window with Custom Client ID:"
for i in {1..3}; do
    echo "Request $i with Client ID 'test-client':"
    curl -s -H "X-Client-ID: test-client" http://localhost:8080/fixed-window
    echo ""
    sleep 1
done

# Test token bucket endpoint
echo -e "\n4. Testing Token Bucket Rate Limiting (5 tokens, 1 token/second refill):"
for i in {1..7}; do
    echo "Request $i:"
    curl -s http://localhost:8080/token-bucket
    echo ""
    sleep 1
done

echo -e "\nTest completed!"
