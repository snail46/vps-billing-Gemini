# Worker Process

The `worker` binary is the asynchronous engine responsible for executing long-running provisioning workflows, transactional outbox publishing, and periodic reconciliation.

## Responsibilities
- Dequeues asynchronous operations from Redis queues.
- Executes multi-step workflows (subscription activation, deterministic node scheduling, resource reservation, virtualization provider calls, IP assignment, state verification).
- Processes and dispatches transactional outbox events.
- Runs periodic background reconciler loops (stuck operation recovery, expired reservation release, node heartbeat monitoring, instance drift detection).

## Running Locally
```bash
go run ./cmd/worker
```
