#!/bin/bash

# E2E test for Slack Slash Commands endpoint
# This script tests the Slack slash commands endpoint with dummy data

set -e

# Track test failures
FAILED_TESTS=0

BASE_URL="${BASE_URL:-http://localhost:8080}"
ENDPOINT="/api/v1/slack/slash"

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

echo "Testing Slack Slash Commands Endpoint: ${BASE_URL}${ENDPOINT}"

# Test /inquiry-help command (without proper signature - should fail)
echo "Testing /inquiry-help command..."
response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "command=/inquiry-help&text=&user_id=U1234567890&channel_id=C1234567890&team_id=T1234567890&response_url=https://hooks.slack.com/commands/1234/5678" \
  -w "%{http_code}" \
  -o /tmp/slash_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ /inquiry-help test PASSED (signature verification working)"
else
    echo "❌ /inquiry-help test FAILED - Expected 401, got HTTP $response"
    cat /tmp/slash_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test /inquiry-status command (without proper signature - should fail)
echo "Testing /inquiry-status command..."
response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "command=/inquiry-status&text=&user_id=U1234567890&channel_id=C1234567890&team_id=T1234567890&response_url=https://hooks.slack.com/commands/1234/5678" \
  -w "%{http_code}" \
  -o /tmp/slash_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ /inquiry-status test PASSED (signature verification working)"
else
    echo "❌ /inquiry-status test FAILED - Expected 401, got HTTP $response"
    cat /tmp/slash_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test unknown command (without proper signature - should fail)
echo "Testing unknown command..."
response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "command=/unknown-command&text=&user_id=U1234567890&channel_id=C1234567890&team_id=T1234567890&response_url=https://hooks.slack.com/commands/1234/5678" \
  -w "%{http_code}" \
  -o /tmp/slash_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ Unknown command test PASSED (signature verification working)"
else
    echo "❌ Unknown command test FAILED - Expected 401, got HTTP $response"
    cat /tmp/slash_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test invalid content type
echo "Testing invalid content type..."
response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d '{"command": "/inquiry-help"}' \
  -w "%{http_code}" \
  -o /tmp/slash_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ Invalid content type test PASSED (signature verification working)"
else
    echo "❌ Invalid content type test FAILED - Expected 401, got HTTP $response"
    cat /tmp/slash_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test /inquiry-help command with VALID signature (should succeed)
echo "Testing /inquiry-help command with valid signature..."
help_payload="command=/inquiry-help&text=&user_id=U1234567890&channel_id=C1234567890&team_id=T1234567890&response_url=https://hooks.slack.com/commands/1234/5678"
timestamp=$(date +%s)
valid_signature=$(generate_slack_signature "$timestamp" "$help_payload")

response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $timestamp" \
  -H "X-Slack-Signature: $valid_signature" \
  -d "$help_payload" \
  -w "%{http_code}" \
  -o /tmp/slash_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 200 ]; then
    echo "✅ Valid signature /inquiry-help test PASSED (200 response)"
else
    echo "❌ Valid signature /inquiry-help test FAILED - Expected 200, got HTTP $response"
    cat /tmp/slash_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test /inquiry-status command with VALID signature (should succeed)
echo "Testing /inquiry-status command with valid signature..."
status_payload="command=/inquiry-status&text=&user_id=U1234567890&channel_id=C1234567890&team_id=T1234567890&response_url=https://hooks.slack.com/commands/1234/5678"
timestamp=$(date +%s)
valid_signature=$(generate_slack_signature "$timestamp" "$status_payload")

response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $timestamp" \
  -H "X-Slack-Signature: $valid_signature" \
  -d "$status_payload" \
  -w "%{http_code}" \
  -o /tmp/slash_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 200 ]; then
    echo "✅ Valid signature /inquiry-status test PASSED (200 response)"
else
    echo "❌ Valid signature /inquiry-status test FAILED - Expected 200, got HTTP $response"
    cat /tmp/slash_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test /inquiry-help command with text parameter and VALID signature (should succeed)
echo "Testing /inquiry-help command with text parameter and valid signature..."
help_text_payload="command=/inquiry-help&text=database&user_id=U1234567890&channel_id=C1234567890&team_id=T1234567890&response_url=https://hooks.slack.com/commands/1234/5678"
timestamp=$(date +%s)
valid_signature=$(generate_slack_signature "$timestamp" "$help_text_payload")

response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $timestamp" \
  -H "X-Slack-Signature: $valid_signature" \
  -d "$help_text_payload" \
  -w "%{http_code}" \
  -o /tmp/slash_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 200 ]; then
    echo "✅ Valid signature /inquiry-help with text test PASSED (200 response)"
else
    echo "❌ Valid signature /inquiry-help with text test FAILED - Expected 200, got HTTP $response"
    cat /tmp/slash_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

echo "All Slack Slash Commands endpoint tests completed"
rm -f /tmp/slash_response.txt

# Exit with error if any tests failed
if [ $FAILED_TESTS -gt 0 ]; then
    echo "❌ $FAILED_TESTS test(s) failed"
    exit 1
fi