# User Web Portal

The `user-web` application is the customer-facing self-service web interface.

## Tech Stack
- React 19, TypeScript, Vite
- Tailwind CSS
- TanStack Query (React Query)
- i18next (Simplified Chinese & English with 100% parity)

## Key Features
- Product catalog exploration with real-time specs (CPU, RAM, Disk, Bandwidth).
- One-click checkout & order preview.
- VPS instance lifecycle controls (start, stop, reboot, reinstall, root password reset).
- Real-time Operation progress drawer via Server-Sent Events (SSE).
- Wallet balance overview and invoice management.
- 6-state UX handling (loading, loaded, empty, error, partial error, permission denied).

## Development
```bash
npm install
npm run dev
```
