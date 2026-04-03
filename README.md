# temporal-serverless-worker

A Go Lambda function that runs as a Temporal serverless worker. The Temporal worker controller invokes this Lambda when there is backlog on a task queue, allowing workers to scale to zero when idle.

See [SETUP.md](./SETUP.md) for full setup instructions.
