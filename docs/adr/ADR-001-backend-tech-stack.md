# ADR-001 — Backend Architecture & Core Technology Stack

Status: Accepted  
Date: 2026-09-23

## Context
The VPS Billing and Automated Resource Management Platform requires a highly reliable, observable, and deterministic backend control plane. As defined in `MASTER_PROMPT.md` and `AGENTS.md`, the platform must guarantee zero silent failures, strict financial consistency, and clear separation of concerns (Order, Payment, Invoice, Subscription, Instance).

## Decision
We select a modular monolith architecture for V1 implemented in Go, utilizing the following stack:
1. **HTTP Routing**: `github.com/go-chi/chi/v5`
   - Lightweight, idiomatic, 100% compliant with standard `net/http`.
   - Rich middleware ecosystem (RequestID, RealIP, Recovery, CORS).
2. **Database Driver & Pool**: `github.com/jackc/pgx/v5` (`pgxpool`)
   - High-performance, native PostgreSQL features (JSONB, UUIDv7, arrays).
   - Connection pool management with idle/lifetime controls.
3. **Data Access Layer**: `sqlc`
   - Compiles idiomatic PostgreSQL queries into type-safe Go code.
   - Zero runtime reflection overhead; compile-time query verification.
   - Explicit SQL prevents ORM magic and unauthorized table scanning.
4. **Database Migrations**: `golang-migrate` / versioned SQL migrations
   - Forward and rollback migrations (`.up.sql` and `.down.sql`).
   - Tracked in `schema_migrations` table; zero unversioned DDL in production.
5. **Caching & Asynchronous Queuing**: `github.com/redis/go-redis/v9`
   - Redis 8-alpine for fast cache and reliable background task queue.
6. **Strict Layered Architecture**:
   - Calling direction: `HTTP Handler -> Application Service -> Domain -> Repository`.
   - Handlers are forbidden from executing raw SQL or directly invoking Repositories.
   - All long-running operations are dispatched as Operations/Workflows to the background Worker.

## Alternatives Considered
- **GORM / Ent**: Rejected due to implicit queries, reflection overhead, and risk of bypassing Ledger transaction boundaries.
- **Microservices with Kafka/gRPC**: Rejected for V1 as unjustified operational complexity; modular monolith provides full isolation with lower operational risk.

## Consequences
- **Positive**: Strict compile-time safety, predictable database access, clear transaction demarcation, fast startup and low memory footprint.
- **Negative**: Requires writing explicit SQL queries and executing code generation via `sqlc generate`.

## Compatibility & Migration
- Schema migrations stored under `db/migrations/` and executed sequentially.
