#!/bin/bash

# Generate sample QR codes for testing
# Usage: ./generate-samples.sh [user-id]

set -e

cd "$(dirname "$0")"

# Load environment variables
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

USER_ID="${1:-sample-user}"

echo "Generating sample QR codes for user: $USER_ID"
echo "Using database: $DATABASE_URL"
echo ""

go run cmd/generate-samples/main.go "$USER_ID"
