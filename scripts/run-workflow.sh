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
NAMESPACE=${NAMESPACE:-default}
TASK_QUEUE=${TQ_NAME:-worker-versioning-sample-ross}

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

echo "Starting $TYPE on $TASK_QUEUE..."
temporal workflow start \
  --type "$TYPE" \
  --task-queue "$TASK_QUEUE" \
  --namespace "$NAMESPACE" \
  --input "$INPUT"
