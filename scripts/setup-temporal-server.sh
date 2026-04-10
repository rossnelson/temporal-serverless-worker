#!/usr/bin/env bash
set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
# Required: path to the temporalio/temporal repo
TEMPORAL_DIR="${TEMPORAL_DIR:?'TEMPORAL_DIR is required. Set it to your local temporalio/temporal repo path.'}"
NAMESPACE="${NAMESPACE:-default}"
IAM_ROLE_ARN="${IAM_ROLE_ARN:?'IAM_ROLE_ARN is required.'}"
LAMBDA_ARN="${LAMBDA_ARN:?'LAMBDA_ARN is required.'}"
EXTERNAL_ID="${EXTERNAL_ID:?'EXTERNAL_ID is required.'}"
: "${AWS_ACCESS_KEY_ID:?'AWS_ACCESS_KEY_ID is required.'}"
: "${AWS_SECRET_ACCESS_KEY:?'AWS_SECRET_ACCESS_KEY is required.'}"

DYNAMIC_CONFIG_RELATIVE="${DYNAMIC_CONFIG_RELATIVE:-config/dynamicconfig/development-sql.yaml}"
DYNAMIC_CONFIG="${TEMPORAL_DIR}/${DYNAMIC_CONFIG_RELATIVE}"

# ── Helpers ───────────────────────────────────────────────────────────────────
log() { echo "[setup-temporal-server] $*"; }

ensure_key() {
  local file="$1" key="$2" block="$3"
  if grep -q "^${key}:" "$file" 2>/dev/null; then
    log "  $key already present, skipping"
  else
    log "  Adding $key"
    printf '\n%s\n' "$block" >> "$file"
  fi
}


# ── Dynamic config ────────────────────────────────────────────────────────────
log "Updating dynamic config: $DYNAMIC_CONFIG"

ensure_key "$DYNAMIC_CONFIG" "workercontroller.enabled" \
"workercontroller.enabled:
  - value: true
    constraints:
      namespace: ${NAMESPACE}"

ensure_key "$DYNAMIC_CONFIG" "workercontroller.compute_providers.enabled" \
"workercontroller.compute_providers.enabled:
  - value:
      - aws-lambda"

ensure_key "$DYNAMIC_CONFIG" "workercontroller.scaling_algorithms.enabled" \
"workercontroller.scaling_algorithms.enabled:
  - value:
      - no-sync"

ensure_key "$DYNAMIC_CONFIG" "workercontroller.compute_providers.aws.require_role_and_external_id" \
"workercontroller.compute_providers.aws.require_role_and_external_id:
  - value: true"

log "Dynamic config updated"

log "Done"
