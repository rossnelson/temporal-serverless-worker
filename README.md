# temporal-serverless-worker

A Go Lambda function that runs as a Temporal serverless worker. The Worker Controller Instance (WCI) invokes this Lambda when there is backlog on a task queue, allowing workers to scale to zero when idle.

See [SETUP.md](./SETUP.md) for one-time AWS and IAM setup instructions.

---

## Getting started

### Prerequisites

- AWS CLI configured (`aws configure`)
- Go 1.24+

### 1. Clone this repo

```bash
git clone https://github.com/temporalio/temporal-serverless-worker
cd temporal-serverless-worker
```

### 2. Create your `.env`

```bash
cp .env.example .env
```

Fill in your values. See `.env.example` for all fields.

### 3. Deploy

Build and upload the worker to Lambda (also updates the Lambda environment variables):

```bash
bash scripts/deploy.sh
```

Re-run whenever the Go code or `.env` config changes.

### 4. Run a workflow

```bash
# Food order (parallel payment + kitchen child workflows)
./scripts/run-workflow.sh food

# Trip booking saga (compensation on failure)
./scripts/run-workflow.sh trip

# Order confirmation (timer + escalation)
./scripts/run-workflow.sh confirm
```

The WCI will invoke the Lambda automatically once there is backlog on the task queue.

---

## Logs

```bash
aws logs tail /aws/lambda/YOUR_FUNCTION_NAME --region us-east-1 --follow
```

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│              Temporal Cloud (staging)            │
│                                                  │
│  ┌─────────────────────────────────────────┐    │
│  │  Worker Controller Instance (WCI)       │    │
│  │  - detects task queue backlog           │    │
│  │  - assumes IAM role via STS             │    │
│  │  - invokes Lambda                       │    │
│  └───────────────────┬─────────────────────┘    │
│                      │                           │
└──────────────────────┼───────────────────────────┘
                       │ invoke
                       ▼
              ┌─────────────────┐
              │   AWS Lambda    │
              │  (Go worker)    │
              └────────┬────────┘
                       │ mTLS
                       ▼
              Temporal Cloud frontend
              (TEMPORAL_ADDRESS)
```

1. The WCI detects backlog on the task queue.
2. It assumes the configured IAM role and invokes the Lambda.
3. Lambda connects to Temporal Cloud via mTLS, polls for tasks, executes workflows and activities, then self-terminates after 60 seconds.

---

## Local dev (ngrok)

To run against a local Temporal server instead of staging, see `scripts/dev.sh`. This requires additional repos and ngrok — set `TEMPORAL_DIR` and `UI_PATH` in `.env`.
