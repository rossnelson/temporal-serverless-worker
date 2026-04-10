#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
UI_DIR="${UI_DIR:-}"

# ── Load env ──────────────────────────────────────────────────────────────────
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
else
  echo "ERROR: $ENV_FILE not found" >&2
  exit 1
fi

: "${TEMPORAL_DIR:?'TEMPORAL_DIR is required in .env'}"
: "${FUNCTION_NAME:?'FUNCTION_NAME is required in .env'}"
: "${REGION:?'REGION is required in .env'}"
: "${TQ_NAME:?'TQ_NAME is required in .env'}"
: "${DEPLOYMENT_NAME:?'DEPLOYMENT_NAME is required in .env'}"
: "${BUILD_ID:?'BUILD_ID is required in .env'}"
: "${AWS_ACCESS_KEY_ID:?'AWS_ACCESS_KEY_ID is required in .env'}"
: "${AWS_SECRET_ACCESS_KEY:?'AWS_SECRET_ACCESS_KEY is required in .env'}"
NAMESPACE="${NAMESPACE:-default}"
UI_DIR="${UI_DIR:-${UI_PATH:?'UI_PATH is required in .env'}}"

export AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_DEFAULT_REGION="${REGION}"

# ── Process tracking ──────────────────────────────────────────────────────────
PIDS=()

cleanup() {
  echo ""
  echo "Shutting down..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
  echo "Done."
}
trap cleanup SIGINT SIGTERM

log() { echo "[dev] $*"; }

# ── Run setup (idempotent — skips changes already applied) ────────────────────
log "Running setup..."
bash "$SCRIPT_DIR/setup-temporal-server.sh" 2>&1 | sed 's/^/[setup] /'
log "Setup complete"

# ── Start Temporal server ─────────────────────────────────────────────────────
log "Starting Temporal server in $TEMPORAL_DIR..."
(
  cd "$TEMPORAL_DIR"
  export AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_REGION="$REGION"
  make start > /tmp/temporal-server.log 2>&1
) &
temporal_pid=$!
PIDS+=($temporal_pid)
log "Temporal server PID: $temporal_pid"

# Wait for Temporal to be ready, then create namespace
log "Waiting for Temporal server to be ready..."
for i in $(seq 1 30); do
  if temporal operator namespace describe "$NAMESPACE" --address localhost:${TEMPORAL_PORT} > /dev/null 2>&1; then
    log "Namespace '$NAMESPACE' already exists"
    break
  fi
  if temporal operator namespace create "$NAMESPACE" --address localhost:${TEMPORAL_PORT} > /dev/null 2>&1; then
    log "Namespace '$NAMESPACE' created"
    break
  fi
  if [[ $i -eq 30 ]]; then
    echo "ERROR: Temporal server did not become ready in time" >&2
    cleanup
    exit 1
  fi
  sleep 1
done

# ── Ensure UI is on the correct branch ───────────────────────────────────────
UI_BRANCH="${UI_BRANCH:-serverless-workers-crud}"
CURRENT_UI_BRANCH="$(git -C "$UI_DIR" rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
if [[ "$CURRENT_UI_BRANCH" != "$UI_BRANCH" ]]; then
  log "UI is on '$CURRENT_UI_BRANCH', switching to '$UI_BRANCH'..."
  git -C "$UI_DIR" checkout "$UI_BRANCH"
else
  log "UI is on '$UI_BRANCH'"
fi

# ── Start UI dev server ───────────────────────────────────────────────────────
log "Starting UI dev server in $UI_DIR..."
(cd "$UI_DIR" && pnpm dev:local-temporal > /tmp/ui-dev.log 2>&1) &
ui_pid=$!
PIDS+=($ui_pid)
log "UI dev server PID: $ui_pid"

# ── Start ngrok ───────────────────────────────────────────────────────────────
log "Starting ngrok TCP tunnel on port 7233..."
ngrok tcp "$TEMPORAL_PORT" --log=stdout > /tmp/ngrok-dev.log 2>&1 &
ngrok_pid=$!
PIDS+=($ngrok_pid)
log "ngrok PID: $ngrok_pid"

# Wait for ngrok tunnel to be established (API up + at least one tunnel)
log "Waiting for ngrok tunnel..."
HOST_PORT=""
for i in $(seq 1 30); do
  HOST_PORT="$(curl -s http://localhost:4040/api/tunnels 2>/dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
tunnels = data.get('tunnels', [])
if tunnels:
    print(tunnels[0]['public_url'].replace('tcp://', ''))
" 2>/dev/null || true)"
  if [[ -n "$HOST_PORT" ]]; then
    break
  fi
  if [[ $i -eq 30 ]]; then
    echo "ERROR: ngrok did not establish a tunnel in time" >&2
    cat /tmp/ngrok-dev.log >&2
    cleanup
    exit 1
  fi
  sleep 1
done
log "ngrok tunnel: $HOST_PORT"

# ── Update Lambda env ─────────────────────────────────────────────────────────
CURRENT_HOST_PORT="$(aws lambda get-function-configuration \
  --function-name "$FUNCTION_NAME" \
  --region "$REGION" \
  --query 'Environment.Variables.HOST_PORT' \
  --output text 2>/dev/null || true)"

if [[ "$CURRENT_HOST_PORT" == "$HOST_PORT" ]]; then
  log "Lambda HOST_PORT unchanged ($HOST_PORT), skipping update"
else
  log "Updating Lambda $FUNCTION_NAME: $CURRENT_HOST_PORT → $HOST_PORT"
  aws lambda update-function-configuration \
    --function-name "$FUNCTION_NAME" \
    --region "$REGION" \
    --environment "Variables={
      HOST_PORT=$HOST_PORT,
      TQ_NAME=$TQ_NAME,
      DEPLOYMENT_NAME=$DEPLOYMENT_NAME,
      BUILD_ID=$BUILD_ID
    }" \
    --output text \
    --query 'FunctionName' > /dev/null
  log "Lambda updated"
fi

# ── Ready ─────────────────────────────────────────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Dev environment ready"
echo "  Temporal:  localhost:${TEMPORAL_PORT}"
echo "  UI:        http://localhost:3000"
echo "  ngrok:     $HOST_PORT"
echo "  Lambda:    $FUNCTION_NAME"
echo ""
echo "  Create a serverless deployment in the UI with these values:"
echo ""
echo "    Deployment name:   $DEPLOYMENT_NAME"
echo "    Task queue:        $TQ_NAME"
echo "    Build ID:          $BUILD_ID"
echo "    Namespace:         $NAMESPACE"
echo "    Lambda ARN:        (see .env)"
echo "    IAM role ARN:      (see .env)"
echo "    External ID:       (see .env)"
echo ""
echo "  Deploy Lambda (required before running workflows):"
echo ""
echo "    bash scripts/deploy.sh"
echo ""
echo "  Run workflows:"
echo ""
echo "    ./scripts/run-workflow.sh food     FoodOrderWorkflow"
echo "    ./scripts/run-workflow.sh trip     TripBookingWorkflow (saga w/ compensation)"
echo "    ./scripts/run-workflow.sh confirm  OrderConfirmationWorkflow (timer + escalation)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Press Ctrl-C to stop all services"
echo ""

wait
