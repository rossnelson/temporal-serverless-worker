#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"

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

WORKFLOW=${1:-""}

# Prefer TEMPORAL_ADDRESS/TEMPORAL_NAMESPACE if set (staging), else local defaults
TEMPORAL_ADDRESS="${TEMPORAL_ADDRESS:-}"
NAMESPACE="${TEMPORAL_NAMESPACE:-${NAMESPACE:-default}}"
TASK_QUEUE="${TQ_NAME:-worker-versioning-sample-ross}"

# Decode base64 TLS certs to temp files and build path flags
TLS_FLAGS=()
TMPDIR_TLS=""
cleanup_tls() { [[ -n "$TMPDIR_TLS" ]] && rm -rf "$TMPDIR_TLS"; }
trap cleanup_tls EXIT

if [[ -n "${TEMPORAL_TLS_CLIENT_CERT_BASE64:-}" && -n "${TEMPORAL_TLS_CLIENT_KEY_BASE64:-}" ]]; then
  TMPDIR_TLS="$(mktemp -d)"
  echo "$TEMPORAL_TLS_CLIENT_CERT_BASE64" | base64 -d > "$TMPDIR_TLS/client.pem"
  echo "$TEMPORAL_TLS_CLIENT_KEY_BASE64"  | base64 -d > "$TMPDIR_TLS/client.key"
  TLS_FLAGS+=(--tls-cert-path "$TMPDIR_TLS/client.pem")
  TLS_FLAGS+=(--tls-key-path  "$TMPDIR_TLS/client.key")
  # Don't set --tls-ca-path: the CLI replaces system root CAs entirely, breaking
  # verification of server certs signed by public CAs. The system trust store handles it.
fi

usage() {
  echo "Usage: $0 <workflow>"
  echo ""
  echo "Workflows:"
  echo "  food     FoodOrderWorkflow         — order food from a restaurant"
  echo "  trip     TripBookingWorkflow       — book flight + hotel + car (saga with compensation)"
  echo "  confirm  OrderConfirmationWorkflow — restaurant confirmation with escalation"
  exit 1
}

case "$WORKFLOW" in
  food)
    TYPE="FoodOrderWorkflow"
    INPUT='{"Customer":"Ross","Restaurant":"Pizzeria Roma","Items":["Margherita Pizza","Caesar Salad","Tiramisu"]}'
    ;;
  trip)
    TYPE="TripBookingWorkflow"
    INPUT='{"Traveler":"Ross","Destination":"Tokyo","DepartureDate":"2026-05-01","ReturnDate":"2026-05-10"}'
    ;;
  confirm)
    TYPE="OrderConfirmationWorkflow"
    INPUT='{"Customer":"Ross","Restaurant":"Pizzeria Roma","OrderID":"ORDER-12345"}'
    ;;
  *)
    usage
    ;;
esac

echo "Starting $TYPE on $TASK_QUEUE (namespace: $NAMESPACE)..."
TEMPORAL_CMD=(temporal workflow start
  --type "$TYPE"
  --task-queue "$TASK_QUEUE"
  --namespace "$NAMESPACE"
  --input "$INPUT"
  "${TLS_FLAGS[@]}"
)
if [[ -n "$TEMPORAL_ADDRESS" ]]; then
  TEMPORAL_CMD+=(--address "$TEMPORAL_ADDRESS")
fi
"${TEMPORAL_CMD[@]}"
