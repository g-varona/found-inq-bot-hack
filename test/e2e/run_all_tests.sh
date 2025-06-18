#!/bin/bash

# E2E Test Runner - Executes all endpoint tests
# This script runs all e2e tests for the Foundation Inquiry Slack Bot

set -e

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_DIR="$(dirname "$0")"

echo "🚀 Running E2E Tests for Foundation Inquiry Slack Bot"
echo "Target URL: ${BASE_URL}"
echo "======================================================="

# Function to check if server is running
check_server() {
    echo "Checking if server is running..."
    if curl -s -f "${BASE_URL}/health" > /dev/null; then
        echo "✅ Server is running"
        return 0
    else
        echo "❌ Server is not running at ${BASE_URL}"
        echo "Please start the server with: make run"
        return 1
    fi
}

# Function to run a test script
run_test() {
    local test_script="$1"
    local test_name="$2"
    
    echo ""
    echo "🧪 Running: $test_name"
    echo "----------------------------------------"
    
    if [ -x "$test_script" ]; then
        if BASE_URL="$BASE_URL" "$test_script"; then
            echo "✅ $test_name completed successfully"
            return 0
        else
            echo "❌ $test_name failed"
            return 1
        fi
    else
        echo "❌ Test script not found or not executable: $test_script"
        return 1
    fi
}

# Make test scripts executable
chmod +x "${TEST_DIR}"/*.sh

# Check if server is running
if ! check_server; then
    exit 1
fi

# Initialize test counters
total_tests=0
passed_tests=0

# Run all tests
echo ""
echo "📋 Test Suite Execution"
echo "======================================================="

# Health endpoint test
total_tests=$((total_tests + 1))
if run_test "${TEST_DIR}/health_test.sh" "Health Endpoint Test"; then
    passed_tests=$((passed_tests + 1))
fi

# Slack Events endpoint test
total_tests=$((total_tests + 1))
if run_test "${TEST_DIR}/slack_events_test.sh" "Slack Events Endpoint Test"; then
    passed_tests=$((passed_tests + 1))
fi

# Slack Slash Commands endpoint test
total_tests=$((total_tests + 1))
if run_test "${TEST_DIR}/slack_slash_test.sh" "Slack Slash Commands Endpoint Test"; then
    passed_tests=$((passed_tests + 1))
fi

# Slack Interactive Components endpoint test
total_tests=$((total_tests + 1))
if run_test "${TEST_DIR}/slack_interactive_test.sh" "Slack Interactive Components Endpoint Test"; then
    passed_tests=$((passed_tests + 1))
fi

# Summary
echo ""
echo "📊 Test Results Summary"
echo "======================================================="
echo "Total Tests: $total_tests"
echo "Passed: $passed_tests"
echo "Failed: $((total_tests - passed_tests))"

if [ $passed_tests -eq $total_tests ]; then
    echo "🎉 All tests PASSED!"
    exit 0
else
    echo "❌ Some tests FAILED!"
    exit 1
fi