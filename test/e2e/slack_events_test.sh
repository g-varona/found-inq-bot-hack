#!/bin/bash

# E2E test for Slack Events API endpoint
# This script tests the Slack Events endpoint with dummy data

set -e

# Track test failures
FAILED_TESTS=0

BASE_URL="${BASE_URL:-http://localhost:8080}"
ENDPOINT="/api/v1/slack/events"

# Function to generate valid Slack signature
generate_slack_signature() {
    local timestamp="$1"
    local body="$2"
    local signing_secret="your-signing-secret-here"
    
    # Create the signature base string: v0:timestamp:body
    local base_string="v0:${timestamp}:${body}"
    
    # Calculate HMAC-SHA256
    local signature=$(echo -n "$base_string" | openssl dgst -sha256 -hmac "$signing_secret" -binary | xxd -p | tr -d '\n')

    echo "v0=${signature}"
}

echo "Testing Slack Events Endpoint: ${BASE_URL}${ENDPOINT}"

# Test URL verification challenge
echo "Testing URL verification challenge..."
url_verification_payload='{"token":"dummy_token","challenge":"3eZbrw1aBm2rZdqrvauRqoZh","type":"url_verification"}'

response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "$url_verification_payload" \
  -w "%{http_code}" \
  -o /tmp/slack_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ URL verification test PASSED (signature verification working)"
else
    echo "❌ URL verification test FAILED - Expected 401, got HTTP $response"
    cat /tmp/slack_response.txt
    echo
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test reaction_added event (without proper signature - should fail)
echo "Testing reaction_added event..."
reaction_payload='{"token":"dummy_token","team_id":"T1234567890","api_app_id":"A1234567890","type":"event_callback","event":{"type":"reaction_added","user":"U1234567890","reaction":"eyes","item":{"type":"message","channel":"C1234567890","ts":"1234567890.123456"},"item_user":"U0987654321","event_ts":"1234567890.123456"}}'

response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "$reaction_payload" \
  -w "%{http_code}" \
  -o /tmp/slack_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ Reaction event test PASSED (signature verification working)"
else
    echo "❌ Reaction event test FAILED - Expected 401, got HTTP $response"
    cat /tmp/slack_response.txt
    echo
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test invalid JSON
echo "Testing invalid JSON..."
response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "{invalid json" \
  -w "%{http_code}" \
  -o /tmp/slack_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ Invalid JSON test PASSED (signature verification working)"
else
    echo "❌ Invalid JSON test FAILED - Expected 401, got HTTP $response"
    cat /tmp/slack_response.txt
    echo
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test URL verification with VALID signature (should succeed)
echo "Testing URL verification with valid signature..."
timestamp=$(date +%s)
valid_signature=$(generate_slack_signature "$timestamp" "$url_verification_payload")

echo "valid signature $valid_signature"

response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Slack-Request-Timestamp: $timestamp" \
  -H "X-Slack-Signature: $valid_signature" \
  -d "$url_verification_payload" \
  -w "%{http_code}" \
  -o /tmp/slack_response.txt \
  "${BASE_URL}${ENDPOINT}")

echo $response


if [ "$response" -eq 200 ]; then
    # Verify response contains the challenge
    challenge_response=$(cat /tmp/slack_response.txt | grep -o '"challenge":"[^"]*"' || echo "")
    if [[ "$challenge_response" == *"3eZbrw1aBm2rZdqrvauRqoZh"* ]]; then
        echo "✅ Valid signature URL verification test PASSED (200 response with correct challenge)"
    else
        echo "❌ Valid signature URL verification test FAILED - Got 200 but incorrect challenge response"
        cat /tmp/slack_response.txt
        echo
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    echo "❌ Valid signature URL verification test FAILED - Expected 200, got HTTP $response "
    cat /tmp/slack_response.txt
    echo
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test reaction_added event with VALID signature (should succeed)
echo "Testing reaction_added event with valid signature..."
timestamp=$(date +%s)
valid_signature=$(generate_slack_signature "$timestamp" "$reaction_payload")

response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Slack-Request-Timestamp: $timestamp" \
  -H "X-Slack-Signature: $valid_signature" \
  -d "$reaction_payload" \
  -w "%{http_code}" \
  -o /tmp/slack_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 200 ]; then
    # Verify response contains status ok
    status_response=$(cat /tmp/slack_response.txt | grep -o '"status":"ok"' || echo "")
    if [[ "$status_response" == *"status\":\"ok"* ]]; then
        echo "✅ Valid signature reaction event test PASSED (200 response with ok status)"
    else
        echo "❌ Valid signature reaction event test FAILED - Got 200 but incorrect status response"
        cat /tmp/slack_response.txt
        echo
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    echo "❌ Valid signature reaction event test FAILED - Expected 200, got HTTP $response"
    cat /tmp/slack_response.txt
    echo
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

echo "All Slack Events endpoint tests completed"
rm -f /tmp/slack_response.txt

# Exit with error if any tests failed
if [ $FAILED_TESTS -gt 0 ]; then
    echo "❌ $FAILED_TESTS test(s) failed"
    exit 1
fi