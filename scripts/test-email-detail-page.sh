#!/bin/bash

# EmailDetailPage Tailwind Refactor - Automated Verification Script
# This script performs automated checks on the refactored code

# Change to project directory
cd "$(dirname "$0")/.."

echo "=========================================="
echo "EmailDetailPage Tailwind Refactor Verification"
echo "=========================================="
echo "Working directory: $(pwd)"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Function to print test results
print_result() {
    local result=$1
    local message=$2

    if [ "$result" -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $message"
        PASSED=$((PASSED + 1))
    elif [ "$result" -eq 1 ]; then
        echo -e "${RED}✗${NC} $message"
        FAILED=$((FAILED + 1))
    else
        echo -e "${YELLOW}⚠${NC} $message"
        WARNINGS=$((WARNINGS + 1))
    fi
}

echo "1. Checking file existence..."
echo "----------------------------"

# Check main component file
if [ -f "zettelkasten-front/src/pages/EmailDetailPage.tsx" ]; then
    print_result 0 "EmailDetailPage.tsx exists"
else
    print_result 1 "EmailDetailPage.tsx not found"
fi

# Check CSS module
if [ -f "zettelkasten-front/src/components/email/EmailContent.module.css" ]; then
    print_result 0 "EmailContent.module.css exists"
else
    print_result 1 "EmailContent.module.css not found"
fi

echo ""
echo "2. Checking for inline styles removal..."
echo "----------------------------------------"

# Check that style injection is removed
if ! grep -q "injectedStyle" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "No injected style objects found"
else
    print_result 1 "Injected style objects still present"
fi

# Check that style prop usage is minimal (only for EmailContent module)
STYLE_PROPS=$(grep -c "style={" zettelkasten-front/src/pages/EmailDetailPage.tsx || true)
if [ "$STYLE_PROPS" -eq 0 ]; then
    print_result 0 "No inline style props found"
else
    print_result 2 "Found $STYLE_PROPS inline style props (may be legitimate)"
fi

echo ""
echo "3. Checking Tailwind class usage..."
echo "-----------------------------------"

# Check for Tailwind classes in header
if grep -q "className=\"px-4 py-2" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Header buttons use Tailwind classes"
else
    print_result 1 "Header buttons missing Tailwind classes"
fi

# Check for Tailwind utility classes
if grep -q "text-gray-" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Text color utilities present"
else
    print_result 1 "Text color utilities missing"
fi

if grep -q "bg-gray-" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Background color utilities present"
else
    print_result 1 "Background color utilities missing"
fi

if grep -q "hover:bg-" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Hover state utilities present"
else
    print_result 1 "Hover state utilities missing"
fi

echo ""
echo "4. Checking CSS module import..."
echo "-------------------------------"

if grep -q "import.*EmailContent.module.css" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "EmailContent CSS module imported"
else
    print_result 1 "EmailContent CSS module not imported"
fi

if grep -q "styles.emailContent" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "EmailContent class used for email body"
else
    print_result 1 "EmailContent class not used"
fi

echo ""
echo "5. Checking responsive classes..."
echo "---------------------------------"

# Check for proper spacing utilities
if grep -q "max-w-2xl" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Max width constraint present"
else
    print_result 1 "Max width constraint missing"
fi

# Check for flex layouts
if grep -q "flex items-center" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Flex layouts used"
else
    print_result 1 "Flex layouts missing"
fi

echo ""
echo "6. Checking TypeScript compilation..."
echo "------------------------------------"

cd zettelkasten-front

# Check TypeScript compilation (dry run)
if npx tsc --noEmit 2>&1 | grep -q "error TS"; then
    print_result 1 "TypeScript compilation has errors"
    echo "Run 'npx tsc --noEmit' to see details"
else
    print_result 0 "TypeScript compilation clean"
fi

cd ..

echo ""
echo "7. Checking for accessibility..."
echo "--------------------------------"

# Check for button elements with proper styling
if grep -q "className=.*cursor-pointer" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Interactive elements have cursor-pointer"
else
    print_result 1 "Interactive elements missing cursor-pointer"
fi

# Check for disabled state styling
if grep -q "cursor-not-allowed" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Disabled states properly styled"
else
    print_result 1 "Disabled states missing proper styling"
fi

echo ""
echo "8. Checking component structure..."
echo "---------------------------------"

# Check for loading state
if grep -q "Loading email\.\.\." zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Loading state present"
else
    print_result 1 "Loading state missing"
fi

# Check for error state
if grep -q "Back to Inbox" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Error state with back button present"
else
    print_result 1 "Error state missing"
fi

# Check for fact extraction dialog
if grep -q "Extracted Facts from Email" zettelkasten-front/src/pages/EmailDetailPage.tsx; then
    print_result 0 "Fact extraction dialog present"
else
    print_result 1 "Fact extraction dialog missing"
fi

echo ""
echo "9. Checking git status..."
echo "------------------------"

cd zettelkasten-front
if git diff --quiet src/pages/EmailDetailPage.tsx; then
    print_result 0 "EmailDetailPage.tsx has no uncommitted changes"
else
    print_result 2 "EmailDetailPage.tsx has uncommitted changes"
fi
cd ..

echo ""
echo "=========================================="
echo "Verification Summary"
echo "=========================================="
echo -e "${GREEN}Passed:${NC} $PASSED"
echo -e "${RED}Failed:${NC} $FAILED"
echo -e "${YELLOW}Warnings:${NC} $WARNINGS"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All critical checks passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Start development server: cd zettelkasten-front && npm start"
    echo "2. Open http://localhost:5173 (or port shown in output)"
    echo "3. Navigate to Emails section and test the detail page"
    echo "4. Verify visual appearance matches original design"
    echo "5. Test all interactive elements"
    echo "6. Check browser console for errors"
    echo "7. Test responsive behavior at different viewport sizes"
    echo ""
    echo "See docs/testing/2026-02-27-email-detail-page-tailwind-verification.md for detailed checklist"
    exit 0
else
    echo -e "${RED}Some checks failed. Please review the issues above.${NC}"
    exit 1
fi
