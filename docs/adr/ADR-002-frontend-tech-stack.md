# ADR-002 — Frontend Architecture & Technology Stack

Status: Accepted  
Date: 2026-09-23

## Context
The platform requires two distinct frontend user experiences:
1. **User Web**: User-centric portal for purchasing, paying, and managing instances. Simple, clear, and reassuring with real-time feedback.
2. **Admin Web**: High-density operational cockpit for monitoring health, diagnosing workflows, tracking finances, managing RBAC, and executing emergency interventions.

Both frontends must be bilingual (`zh-CN` and `en-US`), accessible, responsive, and secure against token leakage.

## Decision
We select a modern TypeScript SPA stack with npm workspaces:
1. **Build Tool & Framework**: Vite + React 19 + TypeScript
   - Instant HMR in development; optimized Rollup chunks in production.
   - Strict TypeScript configuration across all packages.
2. **Styling & Design System**: Tailwind CSS
   - Semantic color system (Green success, Blue info, Yellow warning, Red danger, Gray neutral).
   - Shared design tokens and shared atomic components (`@vps-billing/shared`).
3. **Data Fetching & State Management**: TanStack Query (`@tanstack/react-query`)
   - Declarative data fetching, caching, deduplication, and automated background polling / cache invalidation.
4. **Internationalization**: `i18next` + `react-i18next`
   - Complete coverage of all user-visible text, status codes, error keys, and timestamps.
   - Zero hardcoded English or Chinese in UI code.
5. **Security & Session Model**:
   - Secure HttpOnly Cookie Session auth.
   - Forbidden: Permanent JWT storage in `localStorage`.
   - Complete session separation between User portal and Admin console.
   - CSRF protection on mutating requests (`X-CSRF-Token`).

## Alternatives Considered
- **Single monolithic React application**: Rejected to ensure strict session isolation, independent deployment cycles, and different security boundary postures between users and admins.
- **Client-side JWT in localStorage**: Rejected due to XSS vulnerability and inability to achieve immediate, server-side session revocation.

## Consequences
- **Positive**: Strict session security, clean separation of concerns, reusable shared UI and domain types, seamless bilingual localization.
- **Negative**: Requires maintaining two separate frontend entry points and shared package dependencies.
