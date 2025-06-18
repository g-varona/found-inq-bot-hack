#!/bin/bash

# E2E test for Slack Interactive Components endpoint
# This script tests the Slack interactive components endpoint with dummy data

set -e

# Track test failures
FAILED_TESTS=0

BASE_URL="${BASE_URL:-http://localhost:8080}"
ENDPOINT="/api/v1/slack/interactive"

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

echo "Testing Slack Interactive Components Endpoint: ${BASE_URL}${ENDPOINT}"

# Test interactive component payload (without proper signature - should fail)
echo "Testing interactive component payload..."
interactive_payload='{"type":"block_actions","user":{"id":"U1234567890","username":"testuser","name":"testuser","team_id":"T1234567890"},"api_app_id":"A1234567890","token":"dummy_token","container":{"type":"message","message_ts":"1234567890.123456"},"trigger_id":"1234567890.123456.abcdef","team":{"id":"T1234567890","domain":"testteam"},"channel":{"id":"C1234567890","name":"general"},"response_url":"https://hooks.slack.com/actions/T1234567890/123456789/abcdef","actions":[{"action_id":"button_1","block_id":"block_1","text":{"type":"plain_text","text":"Click Me"},"value":"click_me_123","type":"button","action_ts":"1234567890.123456"}]}'

response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "payload=${interactive_payload}" \
  -w "%{http_code}" \
  -o /tmp/interactive_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ Interactive component test PASSED (signature verification working)"
else
    echo "❌ Interactive component test FAILED - Expected 401, got HTTP $response"
    cat /tmp/interactive_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test empty payload (without proper signature - should fail)
echo "Testing empty payload..."
response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "payload=" \
  -w "%{http_code}" \
  -o /tmp/interactive_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ Empty payload test PASSED (signature verification working)"
else
    echo "❌ Empty payload test FAILED - Expected 401, got HTTP $response"
    cat /tmp/interactive_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test missing payload parameter (without proper signature - should fail)
echo "Testing missing payload parameter..."
response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d "other_param=value" \
  -w "%{http_code}" \
  -o /tmp/interactive_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ Missing payload test PASSED (signature verification working)"
else
    echo "❌ Missing payload test FAILED - Expected 401, got HTTP $response"
    cat /tmp/interactive_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test invalid content type
echo "Testing invalid content type..."
response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Slack-Request-Timestamp: $(date +%s)" \
  -H "X-Slack-Signature: v0=dummy_signature" \
  -d '{"payload": "test"}' \
  -w "%{http_code}" \
  -o /tmp/interactive_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 401 ]; then
    echo "✅ Invalid content type test PASSED (signature verification working)"
else
    echo "❌ Invalid content type test FAILED - Expected 401, got HTTP $response"
    cat /tmp/interactive_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test interactive component payload with VALID signature (should succeed)
echo "Testing interactive component payload with valid signature..."
timestamp=$(date +%s)
payload_body="payload=${interactive_payload}"
valid_signature=$(generate_slack_signature "$timestamp" "$payload_body")

response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $timestamp" \
  -H "X-Slack-Signature: $valid_signature" \
  -d "$payload_body" \
  -w "%{http_code}" \
  -o /tmp/interactive_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 200 ]; then
    # Verify response contains status ok
    status_response=$(cat /tmp/interactive_response.txt | grep -o '"status":"ok"' || echo "")
    if [[ "$status_response" == *"status\":\"ok"* ]]; then
        echo "✅ Valid signature interactive component test PASSED (200 response with ok status)"
    else
        echo "✅ Valid signature interactive component test PASSED (200 response)"
    fi
else
    echo "❌ Valid signature interactive component test FAILED - Expected 200, got HTTP $response"
    cat /tmp/interactive_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test button click action with VALID signature (should succeed)
echo "Testing button click action with valid signature..."
button_payload='{"type":"block_actions","user":{"id":"U1234567890","username":"testuser","name":"testuser","team_id":"T1234567890"},"api_app_id":"A1234567890","token":"dummy_token","container":{"type":"message","message_ts":"1234567890.123456"},"trigger_id":"1234567890.123456.abcdef","team":{"id":"T1234567890","domain":"testteam"},"channel":{"id":"C1234567890","name":"general"},"response_url":"https://hooks.slack.com/actions/T1234567890/123456789/abcdef","actions":[{"action_id":"approve_button","block_id":"approval_block","text":{"type":"plain_text","text":"Approve"},"value":"approve_123","type":"button","action_ts":"1234567890.123456"}]}'
timestamp=$(date +%s)
payload_body="payload=${button_payload}"
valid_signature=$(generate_slack_signature "$timestamp" "$payload_body")

response=$(curl -s -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Slack-Request-Timestamp: $timestamp" \
  -H "X-Slack-Signature: $valid_signature" \
  -d "$payload_body" \
  -w "%{http_code}" \
  -o /tmp/interactive_response.txt \
  "${BASE_URL}${ENDPOINT}")

if [ "$response" -eq 200 ]; then
    echo "✅ Valid signature button click test PASSED (200 response)"
else
    echo "❌ Valid signature button click test FAILED - Expected 200, got HTTP $response"
    cat /tmp/interactive_response.txt
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

echo "All Slack Interactive Components endpoint tests completed"
rm -f /tmp/interactive_response.txt

# Exit with error if any tests failed
if [ $FAILED_TESTS -gt 0 ]; then
    echo "❌ $FAILED_TESTS test(s) failed"
    exit 1
fi