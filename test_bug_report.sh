#!/bin/bash

# Test script for bug report API with GitHub toggle
echo "🧪 Testing Bug Report API with GitHub Toggle..."

# Test 1: Without GitHub configuration (should return 503)
echo ""
echo "📝 Test 1: Bug report WITHOUT GitHub configuration..."
echo "Expected: 503 Service Unavailable"

response=$(curl -s -w "%{http_code}" -X POST http://localhost:8080/api/bug-report \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Bug Report",
    "description": "This should fail without GitHub configuration.",
    "steps": "1. Open the application\n2. Try to submit bug report",
    "expected": "Should return 503 error",
    "browser": "Chrome",
    "os": "Linux",
    "maxPoints": "50",
    "minPoints": "0.5",
    "breakPoint": "50",
    "csvUsed": "Nein",
    "additionalInfo": "This should fail"
  }')

http_code="${response: -3}"
response_body="${response%???}"

echo "HTTP Status: $http_code"
echo "Response: $response_body" | jq '.' 2>/dev/null || echo "Response: $response_body"

if [ "$http_code" = "503" ]; then
    echo "✅ Test 1 PASSED: Correctly rejected without GitHub config"
else
    echo "❌ Test 1 FAILED: Expected 503, got $http_code"
fi

echo ""
echo "💡 To test with GitHub configuration, run:"
echo "export GITHUB_TOKEN='your_personal_access_token'"
echo "export GITHUB_REPO='your-username/notenschluessel'"
echo "Then restart the server and run this script again."

# Test 2: Check if GitHub is configured
if [ -n "$GITHUB_TOKEN" ] && [ -n "$GITHUB_REPO" ]; then
    echo ""
    echo "📝 Test 2: Bug report WITH GitHub configuration..."
    echo "GitHub Token: ${GITHUB_TOKEN:0:10}..."
    echo "GitHub Repo: $GITHUB_REPO"
    
    response2=$(curl -s -w "%{http_code}" -X POST http://localhost:8080/api/bug-report \
      -H "Content-Type: application/json" \
      -d '{
        "title": "Test Bug Report (Configured)",
        "description": "This should work with GitHub configuration.",
        "steps": "1. Configure GitHub\n2. Submit bug report",
        "expected": "Should create GitHub issue",
        "browser": "Chrome",
        "os": "Linux",
        "maxPoints": "50",
        "minPoints": "0.5",
        "breakPoint": "50",
        "csvUsed": "Nein",
        "additionalInfo": "This should create a GitHub issue"
      }')
    
    http_code2="${response2: -3}"
    response_body2="${response2%???}"
    
    echo "HTTP Status: $http_code2"
    echo "Response: $response_body2" | jq '.' 2>/dev/null || echo "Response: $response_body2"
    
    if [ "$http_code2" = "200" ]; then
        echo "✅ Test 2 PASSED: Successfully submitted with GitHub config"
    else
        echo "❌ Test 2 FAILED: Expected 200, got $http_code2"
    fi
else
    echo ""
    echo "ℹ️  Test 2 SKIPPED: GitHub not configured"
fi
