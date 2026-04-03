# Local Temporal + AWS Lambda Serverless Workers: Setup Guide

This guide walks through setting up a local Temporal server that can invoke an AWS Lambda function as a serverless worker. The Lambda connects back to your local Temporal via an ngrok TCP tunnel, polls a task queue, and executes workflows and activities.

**Target audience:** A developer with AWS access and a local `temporalio/temporal` build who has never set up serverless workers before.

---

## Architecture overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Local Machine                           │
│                                                             │
│  ┌──────────────┐     ┌─────────────────┐                  │
│  │  Temporal    │────▶│  Worker         │                  │
│  │  Server      │     │  Controller     │──── invokes ─────┼──▶ AWS Lambda
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
2. It assumes the configured IAM role (with external ID) and invokes Lambda.
3. Lambda starts a Temporal worker process, connecting to your local server via the ngrok TCP address.
4. The worker polls, executes workflows and activities, then self-terminates after 60 seconds.

---

## Prerequisites

- AWS account with IAM permissions to create roles and manage Lambda
- AWS CLI configured (`aws configure`)
- Go 1.24+ installed
- `temporalio/temporal` cloned and buildable locally
- This repo checked out
- ngrok installed and authenticated (`ngrok authtoken YOUR_TOKEN`)

---

## Part 1: AWS setup (one-time)

### 1.1 Create the IAM role

Create a role that the Temporal server will assume to invoke Lambda. In the AWS console or via CLI:

**Role name:** `temporal-assume-role` (or your preferred name)

**Trust policy** — replace `ACCOUNT_ID` and `YOUR_USER` with your actual values:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::ACCOUNT_ID:user/YOUR_USER"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "YOUR_EXTERNAL_ID"
        }
      }
    }
  ]
}
```

The external ID condition is required because `workercontroller.compute_providers.aws.require_role_and_external_id` defaults to `true`. Choose any string for `YOUR_EXTERNAL_ID` — you'll use it again in the UI.

**Permissions policy** — attach an inline policy granting Lambda invocation:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "lambda:InvokeFunction",
      "Resource": "arn:aws:lambda:us-east-1:ACCOUNT_ID:function:YOUR_FUNCTION_NAME"
    }
  ]
}
```

Note the role ARN once created: `arn:aws:iam::ACCOUNT_ID:role/temporal-assume-role`

### 1.2 Verify your IAM user has AssumeRole permission

The user running the Temporal server must be able to assume the role above. Attach this inline policy to your IAM user:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": "arn:aws:iam::ACCOUNT_ID:role/temporal-assume-role"
    }
  ]
}
```

### 1.3 Create the Lambda function

In the AWS console, create a new Lambda function with these settings:

| Setting | Value |
|---------|-------|
| Runtime | `Amazon Linux 2023 (provided.al2023)` |
| Handler | `bootstrap` |
| Architecture | `x86_64` |
| Timeout | `75` seconds |

The timeout must exceed 60 seconds — the worker runs for exactly 1 minute before self-terminating. 75 seconds provides a safe margin.

**Environment variables** (set these now; you'll update `HOST_PORT` each session):

| Variable | Value | Notes |
|----------|-------|-------|
| `HOST_PORT` | `4.tcp.ngrok.io:16955` | Update every session — see Part 3 |
| `TQ_NAME` | `worker-versioning-sample` | Must match task queue in UI/CLI |
| `DEPLOYMENT_NAME` | `test` | Must match deployment name in UI |
| `BUILD_ID` | `v1` | Must match build ID in UI |

Note the function ARN: `arn:aws:lambda:us-east-1:ACCOUNT_ID:function:YOUR_FUNCTION_NAME`

### 1.4 Build and deploy the Go binary

From `lambda-worker/` in this repo:

```bash
cd lambda-worker

# Build for x86_64 Lambda
GOOS=linux GOARCH=amd64 go build -o bootstrap .
zip lambda.zip bootstrap

# Deploy
aws lambda update-function-code \
  --function-name YOUR_FUNCTION_NAME \
  --zip-file fileb://lambda.zip \
  --region us-east-1
```

Alternatively, use the included deploy script. It assumes the role before uploading (useful if your user doesn't have `lambda:UpdateFunctionCode` directly). Edit the `ROLE_ARN` and `FUNCTION_NAME` variables at the top first — the script currently uses `arm64`; change `GOARCH=arm64` to `GOARCH=amd64` if your Lambda is `x86_64`:

```bash
# Review and edit before running
cat deploy.sh

bash deploy.sh
```

---

## Part 2: Temporal server setup (one-time)

### 2.1 Add dynamic config

In your `temporalio/temporal` repo, find the dynamic config file. For SQLite-based local dev, this is typically `config/dynamicconfig/development-sqlite.yaml`. Check `config/development.yaml` for the `dynamicConfigClient.filepath` value to confirm the path.

Add these entries:

```yaml
workercontroller.enabled:
  - value: true
    constraints:
      namespace: default

workercontroller.compute_providers.enabled:
  - value:
      - aws-lambda

workercontroller.scaling_algorithms.enabled:
  - value:
      - no-sync

workercontroller.compute_providers.aws.require_role_and_external_id:
  - value: true
```

The server hot-reloads this file every 10 seconds — no restart needed after editing.

There is also a helper script at `scripts/setup-lambda-worker-config.sh` in the temporal repo that reads credentials from a file, patches the dynamic config idempotently, and prints the environment variables to export before `make start`.

---

## Part 3: Session workflow (every session)

Do these steps in order at the start of each dev session.

### Step 1: Start ngrok

```bash
ngrok tcp 7233
```

Note the forwarding address from the output, for example:

```
Forwarding  tcp://4.tcp.ngrok.io:16955 -> localhost:7233
```

The hostname and port change each session unless you have a paid ngrok plan with reserved addresses.

### Step 2: Update the Lambda environment variable

```bash
aws lambda update-function-configuration \
  --function-name YOUR_FUNCTION_NAME \
  --region us-east-1 \
  --environment "Variables={HOST_PORT=4.tcp.ngrok.io:16955,TQ_NAME=worker-versioning-sample,DEPLOYMENT_NAME=test,BUILD_ID=v1}"
```

Replace `4.tcp.ngrok.io:16955` with the address from Step 1.

### Step 3: Export AWS credentials and start Temporal

The Temporal server process needs AWS credentials in its environment to assume the IAM role. Export them before running `make start` — exporting after the process starts does not work.

```bash
export AWS_ACCESS_KEY_ID=YOUR_KEY
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET
export AWS_DEFAULT_REGION=us-east-1

# From your temporalio/temporal repo
make start
```

### Step 4: Start the UI

From this repo:

```bash
pnpm dev:local-temporal
```

### Step 5: Create a serverless worker deployment in the UI

1. Navigate to **Workers** in the UI
2. Click **Create Serverless Deployment** (or equivalent)
3. Fill in:
   - **Name** — a deployment name (e.g. `test`)
   - **Build ID** — a build identifier (e.g. `v1`)
   - **Lambda ARN** — the full ARN from Part 1.3
   - **IAM Role ARN** — the role ARN from Part 1.1
   - **External ID** — the string you used in the trust policy
4. Submit. The first version is automatically set as current on create.

The `DEPLOYMENT_NAME` and `BUILD_ID` Lambda environment variables must match what you enter here.

---

## Part 4: Test it

### Trigger a workflow

```bash
temporal workflow start \
  --type my-test-workflow \
  --task-queue worker-versioning-sample \
  --namespace default
```

### Watch Lambda logs

```bash
aws logs tail /aws/lambda/YOUR_FUNCTION_NAME \
  --region us-east-1 \
  --since 2m \
  --follow
```

### What success looks like

```
subprocess worker starting  {"hostPort": "4.tcp.ngrok.io:16955", "namespace": "default", "taskQueue": "worker-versioning-sample"}
Started Worker
Running workflow  {"workflow_name": "my-test-workflow"}
Running activity  {"activity_name": "my-test-workflow"}
Worker has been stopped  {"Signal": "timed out"}
```

- `subprocess worker starting` — Lambda connected to Temporal via ngrok
- `Started Worker` — worker registered with the task queue
- `Running workflow` / `Running activity` — workflow executed
- `Worker has been stopped` with `Signal: timed out` — clean exit after 60 seconds

---

## Gotchas

**ngrok address changes every session.**
Always update the Lambda `HOST_PORT` environment variable before starting. If you forget, Lambda will try to connect to a stale address and time out.

**Temporal server must be started with AWS credentials already in the environment.**
The worker controller reads credentials at startup. Setting environment variables after `make start` has no effect on the running process.

**Lambda architecture must match the build target.**
The `bootstrap` binary in this repo is `x86-64`. If you use `deploy.sh` unchanged it builds `arm64` — those two must agree. Either change the Lambda architecture to `arm64` or change the script to `GOARCH=amd64`.

**Lambda timeout must be greater than 60 seconds.**
The worker runs for exactly 1 minute then self-terminates via a `time.AfterFunc(1*time.Minute, ...)` call. With a 60-second timeout Lambda would kill it before it can return cleanly. Use 75 seconds.

**First version must be set as current.**
The UI sets the first version as current automatically on create. If you create additional versions manually via the temporal CLI you need to promote them explicitly:

```bash
temporal worker deployment set-current-version \
  --deployment-name test \
  --build-id v1 \
  --namespace default
```

**`require_role_and_external_id` defaults to true.**
The IAM role trust policy must include the `sts:ExternalId` condition, or the worker controller will reject role assumption attempts. If you want to disable this check for local dev, set the dynamic config value to `false` — but leave it `true` for anything resembling a real environment.

**TLS is disabled by default in this worker.**
The Lambda only enables TLS when the `ENABLE_TLS` environment variable is set. For local Temporal (plaintext gRPC on 7233) this is correct; do not set `ENABLE_TLS` for local dev.

---

## Environment variable reference

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST_PORT` | `127.0.0.1:7233` | Temporal server address. Set to ngrok address for Lambda. |
| `NAMESPACE` | `default` | Temporal namespace |
| `TQ_NAME` | `worker-versioning-sample` | Task queue name |
| `DEPLOYMENT_NAME` | `test` | Worker deployment name (must match UI) |
| `BUILD_ID` | `v1` | Build identifier (must match UI) |
| `ENABLE_TLS` | _(unset)_ | Set to any value to enable TLS (Temporal Cloud) |
| `TLS_CERT` | _(unset)_ | Secrets Manager secret ID for TLS cert (when `ENABLE_TLS` set) |
| `TLS_KEY` | _(unset)_ | Secrets Manager secret ID for TLS key (when `ENABLE_TLS` set) |
| `API_KEY` | _(unset)_ | Secrets Manager secret ID for API key (when `ENABLE_TLS` set) |
