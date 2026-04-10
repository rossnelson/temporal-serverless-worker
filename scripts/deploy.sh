#!/usr/bin/env bash
set -euo pipefail

# Load env vars from .env if present
if [[ -f "$(dirname "$0")/../.env" ]]; then
  # shellcheck source=/dev/null
  source "$(dirname "$0")/../.env"
fi

ROLE_ARN="${ROLE_ARN:?'ROLE_ARN is required. Set it in .env or your environment.'}"
FUNCTION_NAME="${FUNCTION_NAME:-temporal-serverless-worker}"
REGION="${REGION:-us-east-1}"
LAMBDA_ARCH="${LAMBDA_ARCH:-arm64}"
ZIP_FILE="lambda.zip"

echo "Building..."
GOOS=linux GOARCH="$LAMBDA_ARCH" go build -o bootstrap .
zip -q "$ZIP_FILE" bootstrap

echo "Deploying to $FUNCTION_NAME..."
aws lambda update-function-code \
  --function-name "$FUNCTION_NAME" \
  --zip-file "fileb://$ZIP_FILE" \
  --region "$REGION"

echo "Done."
