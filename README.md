# temporal-serverless-worker

A Go Lambda function that runs as a Temporal serverless worker. The Temporal worker controller invokes this Lambda when there is backlog on a task queue, allowing workers to scale to zero when idle.

See [SETUP.md](./SETUP.md) for one-time AWS and IAM setup instructions.

---

## Getting started

### Prerequisites

- AWS CLI configured (`aws configure`)
- Go 1.24+
- Node.js + pnpm
- ngrok installed and authenticated (`ngrok authtoken YOUR_TOKEN`)

### 1. Clone the repos

```bash
# This repo
git clone https://github.com/temporalio/temporal-serverless-worker
cd temporal-serverless-worker

# Temporal server
git clone https://github.com/temporalio/temporal ~/code/temporal

# Temporal UI (or wherever you keep it)
git clone https://github.com/temporalio/ui ~/code/ui
```

### 2. Create your `.env`

Copy the template below into `.env` at the root of this repo and fill in your values.

```bash
# AWS credentials
AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_SECRET_ACCESS_KEY

# deploy.sh
ROLE_ARN=arn:aws:iam::ACCOUNT_ID:role/YOUR_ROLE
FUNCTION_NAME=YOUR_LAMBDA_FUNCTION_NAME
REGION=us-east-1

# setup-temporal-server.sh
TEMPORAL_DIR=/path/to/temporalio/temporal
NAMESPACE=default
IAM_ROLE_ARN=arn:aws:iam::ACCOUNT_ID:role/YOUR_ROLE
LAMBDA_ARN=arn:aws:lambda:us-east-1:ACCOUNT_ID:function:YOUR_LAMBDA_FUNCTION_NAME
EXTERNAL_ID=YOUR_EXTERNAL_ID

# Worker config
TQ_NAME=worker-versioning-sample
DEPLOYMENT_NAME=my-deployment
BUILD_ID=v1

# Local paths
UI_PATH=/path/to/temporalio/ui
```

> `IAM_ROLE_ARN` and `ROLE_ARN` are the same role — one is used by the Temporal server to invoke Lambda, the other by `deploy.sh` to upload the binary.

### 3. Deploy the Lambda binary

Build and upload the worker to Lambda:

```bash
bash scripts/deploy.sh
```

This only needs to be re-run when the Go code changes.

### 4. Start the dev environment

```bash
./scripts/dev.sh
```

This single command:
- Runs one-time Temporal server configuration (idempotent — safe to run every session)
- Starts the Temporal server
- Starts the UI dev server
- Starts an ngrok TCP tunnel on port 7233
- Updates the Lambda `HOST_PORT` env var if the ngrok address changed
- Creates the `default` namespace if it doesn't exist

When ready, it prints the values you need to create a serverless deployment in the UI.

### 5. Create a deployment in the UI

Open [http://localhost:3000](http://localhost:3000), navigate to **Workers**, and create a new serverless deployment using the values printed by `dev.sh`.

### 6. Run a workflow

```bash
# Food order (parallel payment + kitchen child workflows)
./scripts/run-workflow.sh food

# Trip booking saga (compensation on failure)
./scripts/run-workflow.sh trip

# Order confirmation (timer + escalation)
./scripts/run-workflow.sh confirm
```

---

## Logs

```bash
# Temporal server
tail -f /tmp/temporal-server.log

# UI dev server
tail -f /tmp/ui-dev.log

# Lambda
aws logs tail /aws/lambda/YOUR_FUNCTION_NAME --region us-east-1 --follow
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Local Machine                           │
│                                                             │
│  ┌──────────────┐     ┌─────────────────┐                  │
│  │  Temporal    │────▶│  Worker         │                  │
│  │  Server      │     │  Controller     │── invokes ───────┼──▶ AWS Lambda
│  │  :7233       │     │  (watches TQ)   │                  │       │
│  └──────┬───────┘     └─────────────────┘                  │       │
│         │                                                   │       │
│  ┌──────▼───────┐                                          │       │
│  │    ngrok     │◀─────────────────────────────────────────┼───────┘
│  │  TCP tunnel  │   Lambda connects back via ngrok          │
│  └──────────────┘                                           │
└─────────────────────────────────────────────────────────────┘
```

1. The worker controller detects backlog on the task queue.
2. It assumes the configured IAM role and invokes Lambda.
3. Lambda starts a Temporal worker, connecting back via the ngrok TCP address.
4. The worker polls, executes workflows and activities, then self-terminates after 60 seconds.
