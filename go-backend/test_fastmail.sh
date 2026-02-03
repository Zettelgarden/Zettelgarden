#!/bin/bash
# Test script to verify Fastmail CalDAV authentication
# Usage: ./test_fastmail.sh <email> <app-password> <calendar-url>

EMAIL="$1"
PASSWORD="$2"
CALENDAR_URL="$3"

if [ -z "$EMAIL" ] || [ -z "$PASSWORD" ] || [ -z "$CALENDAR_URL" ]; then
    echo "Usage: $0 <email> <app-password> <calendar-url>"
    echo "Example: $0 nick@example.com abcdefghijklmnop https://caldav.fastmail.com/dav/calendars/user/nick@example.com/UUID"
    exit 1
fi

echo "=== Testing Fastmail CalDAV Authentication ==="
echo "Email: $EMAIL"
echo "URL: $CALENDAR_URL"
echo "Password length: ${#PASSWORD}"
echo ""

# Test 1: Simple GET with basic auth
echo "Test 1: Simple GET with basic auth"
curl -v -u "$EMAIL:$PASSWORD" \
    -H "User-Agent: Zettelgarden/1.0" \
    -H "Accept: text/calendar, application/octet-stream" \
    "$CALENDAR_URL" 2>&1 | head -50

echo ""
echo "=== Done ==="
