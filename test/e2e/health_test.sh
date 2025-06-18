#!/bin/bash

# E2E test for health endpoint
# This script tests the health check endpoint

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
ENDPOINT="/health"

echo "Testing Health Endpoint: ${BASE_URL}${ENDPOINT}"

# Test health endpoint
response=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 200 ]; then
    echo "✅ Health endpoint test PASSED"
    # Get and display the response body
    curl -s "${BASE_URL}${ENDPOINT}" | jq '.'
else
    echo "❌ Health endpoint test FAILED - HTTP $response"
    exit 1
fi