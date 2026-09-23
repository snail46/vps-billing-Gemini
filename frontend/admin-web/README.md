# Admin Web Console

The `admin-web` application is the administrative management dashboard for operators and platform owners.

## Tech Stack
- React 19, TypeScript, Vite
- Tailwind CSS
- TanStack Query (React Query)
- i18next (Simplified Chinese & English with 100% parity)

## Key Features
- Platform health status & system metrics overview.
- Infrastructure management: virtualization providers, node groups, compute nodes, and IP pools.
- Double-entry accounting ledger inspection and financial audit trails.
- Order & subscription oversight.
- Asynchronous Operation monitoring with manual retry and cancellation capabilities.
- Admin user & role-based access control (RBAC).
- RFC 6238 TOTP two-factor authentication (2FA) setup and enforcement.

## Development
```bash
npm install
npm run dev
```
