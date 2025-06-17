#!/bin/bash

# Test script for bug report API
echo "🧪 Testing Bug Report API..."

# Test without GitHub configuration (should work with console logging)
echo "📝 Sending test bug report..."

curl -X POST http://localhost:8080/api/bug-report \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Bug Report",
    "description": "This is a test bug report to verify the API works correctly.",
    "steps": "1. Open the application\n2. Click the bug report button\n3. Fill out the form",
    "expected": "The bug report should be submitted successfully",
    "browser": "Chrome",
    "os": "Linux",
    "maxPoints": "50",
    "minPoints": "0.5",
    "breakPoint": "50",
    "csvUsed": "Nein",
    "additionalInfo": "This is an automated test"
  }' \
  | jq '.'

echo ""
echo "✅ Test completed! Check the server logs for the bug report entry."
echo ""
echo "💡 To enable GitHub integration:"
echo "export GITHUB_TOKEN='your_personal_access_token'"
echo "export GITHUB_REPO='your-username/notenschluessel'"
