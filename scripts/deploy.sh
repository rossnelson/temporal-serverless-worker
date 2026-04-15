#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"

# Load env vars from .env if present
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

: "${LAMBDA_ARN:?'LAMBDA_ARN is required. Set it in .env or your environment.'}"
FUNCTION_NAME="${FUNCTION_NAME:-${LAMBDA_ARN##*:function:}}"
REGION="${REGION:-us-east-1}"
LAMBDA_ARCH="${LAMBDA_ARCH:-amd64}"
ZIP_FILE="$ROOT_DIR/lambda.zip"

echo "Building..."
cd "$ROOT_DIR"
GOOS=linux GOARCH="$LAMBDA_ARCH" go build -o bootstrap .
zip -q "$ZIP_FILE" bootstrap

echo "Deploying code to $FUNCTION_NAME..."
aws lambda update-function-code \
  --function-name "$FUNCTION_NAME" \
  --zip-file "fileb://$ZIP_FILE" \
  --region "$REGION" \
  --output text --query 'FunctionName' > /dev/null

echo "Waiting for code update to complete..."
aws lambda wait function-updated \
  --function-name "$FUNCTION_NAME" \
  --region "$REGION"

# ── Build Lambda environment variables ───────────────────────────────────────
# Prefer TEMPORAL_ADDRESS/TEMPORAL_NAMESPACE if set, else fall back to HOST_PORT/NAMESPACE
HOST_PORT_VAL="${TEMPORAL_ADDRESS:-${HOST_PORT:-}}"
NAMESPACE_VAL="${TEMPORAL_NAMESPACE:-${NAMESPACE:-default}}"

ENV_VARS="TQ_NAME=${TQ_NAME:?},DEPLOYMENT_NAME=${DEPLOYMENT_NAME:?},BUILD_ID=${BUILD_ID:?},NAMESPACE=$NAMESPACE_VAL"

if [[ -n "$HOST_PORT_VAL" ]]; then
  ENV_VARS="$ENV_VARS,HOST_PORT=$HOST_PORT_VAL"
fi

# Staging: pass base64-encoded TLS certs directly
if [[ -n "${TEMPORAL_TLS_CLIENT_CERT_BASE64:-}" ]]; then
  ENV_VARS="$ENV_VARS,TEMPORAL_TLS_CLIENT_CERT_BASE64=$TEMPORAL_TLS_CLIENT_CERT_BASE64"
fi
if [[ -n "${TEMPORAL_TLS_CLIENT_KEY_BASE64:-}" ]]; then
  ENV_VARS="$ENV_VARS,TEMPORAL_TLS_CLIENT_KEY_BASE64=$TEMPORAL_TLS_CLIENT_KEY_BASE64"
fi
if [[ -n "${TEMPORAL_TLS_SERVER_ROOT_CA_CERT_BASE64:-}" ]]; then
  ENV_VARS="$ENV_VARS,TEMPORAL_TLS_SERVER_ROOT_CA_CERT_BASE64=$TEMPORAL_TLS_SERVER_ROOT_CA_CERT_BASE64"
fi

echo "Updating Lambda config: $FUNCTION_NAME"
aws lambda update-function-configuration \
  --function-name "$FUNCTION_NAME" \
  --region "$REGION" \
  --environment "Variables={$ENV_VARS}" \
  --output text --query 'FunctionName' > /dev/null

echo "Done. Lambda targeting: ${HOST_PORT_VAL:-<unchanged>} / namespace: $NAMESPACE_VAL"
