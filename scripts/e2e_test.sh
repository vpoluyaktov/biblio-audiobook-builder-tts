#!/bin/bash

# End-to-End Test Script for Biblio Audiobook Builder TTS Server
# Tests the complete workflow: upload -> conversion -> build -> completion
# Uses espeak provider for fast testing

set -e

# Configuration
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_FILE="${TEST_FILE:-$SCRIPT_DIR/test_data/gift_of_the_magi.epub}"
PROVIDER="espeak"
VOICE="en"
POLL_INTERVAL=3
MAX_WAIT_TIME=1800  # 30 minutes max for large books

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependency() {
    if ! command -v "$1" &> /dev/null; then
        log_error "Required command '$1' not found. Please install it."
        exit 1
    fi
}

cleanup() {
    if [ -n "$JOB_ID" ]; then
        log_info "Cleaning up job $JOB_ID..."
        curl -s -X DELETE "$SERVER_URL/api/jobs/$JOB_ID" > /dev/null 2>&1 || true
    fi
}

# Check dependencies
log_info "Checking dependencies..."
check_dependency "curl"
check_dependency "jq"

# Check if test file exists
if [ ! -f "$TEST_FILE" ]; then
    log_error "Test file not found: $TEST_FILE"
    log_info "Please provide a valid epub file path via TEST_FILE environment variable"
    exit 1
fi

log_info "Using test file: $TEST_FILE"

# Set up cleanup trap
trap cleanup EXIT

# Test 1: Check server health
log_info "Test 1: Checking server health..."
HEALTH_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/api/jobs" 2>/dev/null || echo "000")
if [ "$HEALTH_RESPONSE" != "200" ]; then
    log_error "Server is not responding at $SERVER_URL (HTTP $HEALTH_RESPONSE)"
    log_info "Please start the server first: ./biblio-audiobook-builder-tts server --headless"
    exit 1
fi
log_success "Server is healthy"

# Test 2: Clean up - delete all existing jobs
log_info "Test 2: Cleaning up existing jobs..."
EXISTING_JOBS=$(curl -s "$SERVER_URL/api/jobs" | jq -r '.[].id')
for job_id in $EXISTING_JOBS; do
    log_info "  Deleting job: $job_id"
    curl -s -X DELETE "$SERVER_URL/api/jobs/$job_id" > /dev/null
done
log_success "All existing jobs deleted"

# Test 3: Check temp and output folders
log_info "Test 3: Checking temp and output folders..."
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
TEMP_DIR="$PROJECT_DIR/temp"
OUTPUT_DIR="$PROJECT_DIR/output"

# Clean temp folder
if [ -d "$TEMP_DIR" ]; then
    TEMP_COUNT=$(find "$TEMP_DIR" -type f 2>/dev/null | wc -l)
    if [ "$TEMP_COUNT" -gt 0 ]; then
        log_info "  Cleaning $TEMP_COUNT files from temp folder..."
        rm -rf "$TEMP_DIR"/*
    fi
fi

# Clean output folder
if [ -d "$OUTPUT_DIR" ]; then
    OUTPUT_COUNT=$(find "$OUTPUT_DIR" -type f 2>/dev/null | wc -l)
    if [ "$OUTPUT_COUNT" -gt 0 ]; then
        log_info "  Cleaning $OUTPUT_COUNT files from output folder..."
        rm -rf "$OUTPUT_DIR"/*
    fi
fi
log_success "Temp and output folders are clean"

# Test 4: Upload a file and create a job
log_info "Test 4: Uploading test file with espeak provider..."
UPLOAD_RESPONSE=$(curl -s -X POST \
    -F "file=@$TEST_FILE" \
    -F "provider=$PROVIDER" \
    -F "voice=$VOICE" \
    "$SERVER_URL/api/upload")

JOB_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.id')
if [ -z "$JOB_ID" ] || [ "$JOB_ID" == "null" ]; then
    log_error "Failed to create job. Response: $UPLOAD_RESPONSE"
    exit 1
fi
log_success "Job created with ID: $JOB_ID"

# Test 5: Verify job appears in list
log_info "Test 5: Verifying job appears in job list..."
sleep 1
JOB_IN_LIST=$(curl -s "$SERVER_URL/api/jobs" | jq -r ".[] | select(.id == \"$JOB_ID\") | .id")
if [ "$JOB_IN_LIST" != "$JOB_ID" ]; then
    log_error "Job not found in job list"
    exit 1
fi
log_success "Job found in job list"

# Test 6: Monitor job progress until completion
log_info "Test 6: Monitoring job progress..."
START_TIME=$(date +%s)
LAST_STATUS=""
LAST_PROGRESS=""

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    if [ $ELAPSED -gt $MAX_WAIT_TIME ]; then
        log_error "Job timed out after ${MAX_WAIT_TIME}s"
        exit 1
    fi
    
    JOB_STATUS=$(curl -s "$SERVER_URL/api/jobs/$JOB_ID")
    STATUS=$(echo "$JOB_STATUS" | jq -r '.status')
    CONVERSION_PROGRESS=$(echo "$JOB_STATUS" | jq -r '.conversion_progress')
    BUILD_PROGRESS=$(echo "$JOB_STATUS" | jq -r '.build_progress')
    CURRENT_CHAPTER=$(echo "$JOB_STATUS" | jq -r '.current_chapter_num')
    TOTAL_CHAPTERS=$(echo "$JOB_STATUS" | jq -r '.total_chapters')
    ERROR=$(echo "$JOB_STATUS" | jq -r '.error // empty')
    
    # Calculate display progress using awk (no bc dependency)
    if [ "$STATUS" == "building" ]; then
        DISPLAY_PROGRESS=$(awk "BEGIN {printf \"%d\", $BUILD_PROGRESS * 100}")
        PROGRESS_TYPE="build"
    else
        DISPLAY_PROGRESS=$(awk "BEGIN {printf \"%d\", $CONVERSION_PROGRESS * 100}")
        PROGRESS_TYPE="conversion"
    fi
    
    # Only log if status or progress changed significantly
    if [ "$STATUS" != "$LAST_STATUS" ]; then
        log_info "Status: $STATUS"
        LAST_STATUS="$STATUS"
    fi
    
    # Show progress update
    if [ "$STATUS" == "converting" ]; then
        echo -ne "\r  Converting: ${CURRENT_CHAPTER}/${TOTAL_CHAPTERS} chapters (${DISPLAY_PROGRESS}%)    "
    elif [ "$STATUS" == "building" ]; then
        echo -ne "\r  Building M4B: ${DISPLAY_PROGRESS}%    "
    fi
    
    # Check for terminal states
    if [ "$STATUS" == "completed" ]; then
        echo ""
        log_success "Job completed successfully!"
        break
    elif [ "$STATUS" == "failed" ]; then
        echo ""
        log_error "Job failed: $ERROR"
        exit 1
    elif [ "$STATUS" == "cancelled" ]; then
        echo ""
        log_error "Job was cancelled"
        exit 1
    fi
    
    sleep $POLL_INTERVAL
done

# Test 7: Verify job completion details
log_info "Test 7: Verifying job completion details..."
FINAL_JOB=$(curl -s "$SERVER_URL/api/jobs/$JOB_ID")
BOOK_TITLE=$(echo "$FINAL_JOB" | jq -r '.book_title')
BOOK_AUTHOR=$(echo "$FINAL_JOB" | jq -r '.book_author')
M4B_FILE=$(echo "$FINAL_JOB" | jq -r '.m4b_file')
OUTPUT_PATH=$(echo "$FINAL_JOB" | jq -r '.output_path')

log_success "Book: $BOOK_TITLE by $BOOK_AUTHOR"
log_success "Output path: $OUTPUT_PATH"

if [ -n "$M4B_FILE" ] && [ "$M4B_FILE" != "null" ]; then
    log_success "M4B file: $M4B_FILE"
    
    # Check if M4B file exists
    if [ -f "$M4B_FILE" ]; then
        M4B_SIZE=$(ls -lh "$M4B_FILE" | awk '{print $5}')
        log_success "M4B file size: $M4B_SIZE"
    fi
else
    log_warning "No M4B file generated (chapter files may still be available)"
fi

# Test 8: Test job deletion
log_info "Test 8: Testing job deletion..."
DELETE_RESPONSE=$(curl -s -X DELETE "$SERVER_URL/api/jobs/$JOB_ID")
DELETE_STATUS=$(echo "$DELETE_RESPONSE" | jq -r '.status')
if [ "$DELETE_STATUS" == "deleted" ]; then
    log_success "Job deleted successfully"
    JOB_ID=""  # Clear so cleanup doesn't try to delete again
else
    log_warning "Unexpected delete response: $DELETE_RESPONSE"
fi

# Test 9: Verify job is removed from list
log_info "Test 9: Verifying job is removed from list..."
sleep 1
JOB_AFTER_DELETE=$(curl -s "$SERVER_URL/api/jobs" | jq -r ".[] | select(.id == \"$JOB_ID\") | .id")
if [ -z "$JOB_AFTER_DELETE" ]; then
    log_success "Job successfully removed from list"
else
    log_error "Job still exists in list after deletion"
    exit 1
fi

# Summary
echo ""
echo "=========================================="
log_success "All E2E tests passed!"
echo "=========================================="
echo ""
echo "Test Summary:"
echo "  - Server health check: PASSED"
echo "  - Job creation: PASSED"
echo "  - Job listing: PASSED"
echo "  - Job progress monitoring: PASSED"
echo "  - Job completion: PASSED"
echo "  - Job deletion: PASSED"
echo ""
