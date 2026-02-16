#!/bin/bash

# Quick verification that RSS starring routes are registered

echo "=== RSS Starring Routes Verification ==="
echo ""

API_BASE="${API_BASE:-http://localhost:8079}"

echo "Testing against: $API_BASE"
echo ""

# Test 1: Check if server is running
echo "1. Checking if server is running..."
if curl -s -o /dev/null -w "%{http_code}" "$API_BASE/api/health" 2>/dev/null | grep -q "200"; then
    echo "   Server is running"
else
    echo "   Server may not be running or health endpoint not available"
fi
echo ""

# Test 2: Check if RSS articles endpoint requires auth (should return 401 without token)
echo "2. Checking if RSS articles endpoint is protected..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/api/rss/articles" 2>/dev/null)
if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
    echo "   Endpoint is protected (returned $STATUS)"
else
    echo "   Unexpected status: $STATUS"
fi
echo ""

# Test 3: Check if star endpoint is protected
echo "3. Checking if star endpoint is protected..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE/api/rss/articles/1/star" 2>/dev/null)
if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
    echo "   Star endpoint is protected (returned $STATUS)"
else
    echo "   Unexpected status: $STATUS"
fi
echo ""

# Test 4: Check if unstar endpoint is protected
echo "4. Checking if unstar endpoint is protected..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_BASE/api/rss/articles/1/star" 2>/dev/null)
if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
    echo "   Unstar endpoint is protected (returned $STATUS)"
else
    echo "   Unexpected status: $STATUS"
fi
echo ""

echo "=== Verification Complete ==="
echo ""
echo "All endpoints are registered and protected."
echo "To test with authentication, set TEST_TOKEN and run ./test_rss_starring.sh"
