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

echo "Assuming role $ROLE_ARN..."
creds=$(aws sts assume-role \
  --role-arn "$ROLE_ARN" \
  --role-session-name deploy \
  --output json)

export AWS_ACCESS_KEY_ID=$(echo "$creds" | python3 -c "import sys,json; print(json.load(sys.stdin)['Credentials']['AccessKeyId'])")
export AWS_SECRET_ACCESS_KEY=$(echo "$creds" | python3 -c "import sys,json; print(json.load(sys.stdin)['Credentials']['SecretAccessKey'])")
export AWS_SESSION_TOKEN=$(echo "$creds" | python3 -c "import sys,json; print(json.load(sys.stdin)['Credentials']['SessionToken'])")

echo "Deploying to $FUNCTION_NAME..."
aws lambda update-function-code \
  --function-name "$FUNCTION_NAME" \
  --zip-file "fileb://$ZIP_FILE" \
  --region "$REGION"

echo "Done."
