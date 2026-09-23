# Server Process

The `server` binary hosts the platform's core HTTP REST API, SSE streaming endpoints, and health monitoring probes.

## Responsibilities
- Public catalog, orders, subscriptions, and wallet management APIs.
- Payment webhooks with HMAC-SHA256 signature verification and atomic idempotency deduplication.
- Instance control actions (start, stop, reboot, reinstall, reset password) returning HTTP 202 Accepted with Operation IDs.
- Admin management APIs with session authentication and role-based access control (RBAC).
- RFC 6238 TOTP two-factor authentication (2FA).
- Real-time Server-Sent Events (SSE) progress streams (`/api/v1/events`).
- Prometheus metrics (`/metrics`) and health checks (`/health/live`, `/health/ready`).

## Running Locally
```bash
go run ./cmd/server
```
