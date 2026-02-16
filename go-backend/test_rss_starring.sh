#!/bin/bash

# Manual test script for RSS article starring feature
# This script tests the RSS article starring API endpoints

set -e

echo "=== RSS Article Starring Feature Test ==="
echo ""

# Configuration
API_BASE="${API_BASE:-http://localhost:8079}"
USER_ID="${USER_ID:-1}"

echo "Testing against: $API_BASE"
echo "Using User ID: $USER_ID"
echo ""

# Function to generate a test JWT token
generate_token() {
    # This uses the SECRET_KEY from the environment
    # For testing, you might need to use an existing token or login first
    if [ -n "$TEST_TOKEN" ]; then
        echo "$TEST_TOKEN"
        return
    fi

    echo "Error: No TEST_TOKEN found. Please set TEST_TOKEN environment variable."
    echo "You can get a token by logging in via the API or using an existing session."
    exit 1
}

TOKEN=$(generate_token)

echo "Using token: ${TOKEN:0:20}..."
echo ""

# Test 1: Get articles list
echo "Test 1: Get articles list"
echo "GET $API_BASE/api/rss/articles"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -H "Authorization: Bearer $TOKEN" \
    "$API_BASE/api/rss/articles" \
    -o /tmp/articles_response.json
echo ""

# Parse response to get an article ID
ARTICLE_ID=$(cat /tmp/articles_response.json | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

if [ -z "$ARTICLE_ID" ]; then
    echo "No articles found. Creating test data first..."
    # Try to create a test feed and fetch articles
    echo "Creating test RSS feed..."
    curl -s -X POST \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"url":"https://example.com/feed.xml","name":"Test Feed"}' \
        "$API_BASE/api/rss/feeds" \
        -o /tmp/feed_response.json

    # Try to get articles again
    curl -s -H "Authorization: Bearer $TOKEN" \
        "$API_BASE/api/rss/articles" \
        -o /tmp/articles_response.json

    ARTICLE_ID=$(cat /tmp/articles_response.json | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
fi

if [ -z "$ARTICLE_ID" ]; then
    echo "Still no articles found. Exiting."
    exit 1
fi

echo "Found article ID: $ARTICLE_ID"
echo ""

# Test 2: Star an article
echo "Test 2: Star an article"
echo "POST $API_BASE/api/rss/articles/$ARTICLE_ID/star"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -X POST \
    -H "Authorization: Bearer $TOKEN" \
    "$API_BASE/api/rss/articles/$ARTICLE_ID/star"
echo ""

# Test 3: Get starred articles
echo "Test 3: Get starred articles"
echo "GET $API_BASE/api/rss/articles?starred=true"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -H "Authorization: Bearer $TOKEN" \
    "$API_BASE/api/rss/articles?starred=true" \
    -o /tmp/starred_response.json
echo ""

# Verify the article is in starred list
if grep -q "\"id\":$ARTICLE_ID" /tmp/starred_response.json; then
    echo "SUCCESS: Article $ARTICLE_ID is in starred list"
else
    echo "FAILED: Article $ARTICLE_ID not found in starred list"
fi
echo ""

# Test 4: Unstar the article
echo "Test 4: Unstar the article"
echo "DELETE $API_BASE/api/rss/articles/$ARTICLE_ID/star"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -X DELETE \
    -H "Authorization: Bearer $TOKEN" \
    "$API_BASE/api/rss/articles/$ARTICLE_ID/star"
echo ""

# Test 5: Verify article is unstarred
echo "Test 5: Get starred articles (should be empty or not contain our article)"
echo "GET $API_BASE/api/rss/articles?starred=true"
curl -s -w "\nHTTP Status: %{http_code}\n" \
    -H "Authorization: Bearer $TOKEN" \
    "$API_BASE/api/rss/articles?starred=true" \
    -o /tmp/starred_after_unstar.json
echo ""

if grep -q "\"id\":$ARTICLE_ID" /tmp/starred_after_unstar.json; then
    echo "FAILED: Article $ARTICLE_ID still in starred list after unstar"
else
    echo "SUCCESS: Article $ARTICLE_ID removed from starred list"
fi
echo ""

echo "=== Tests Complete ==="
