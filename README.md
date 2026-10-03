# SchoolHub

A full-stack school administration & management portal, built as a portfolio
prototype demonstrating practical software engineering: Next.js/TypeScript on
the frontend, Go on the backend, PostgreSQL for storage.

> **Status:** Milestone 5 — RBAC/permissions. Project scaffolding, PostgreSQL
> migrations, the sqlc data layer, session authentication, and permission-based
> authorization are implemented and verified. Domain modules (staff, students,
> academics, attendance, results, ...) arrive in later milestones.

**This is a personal portfolio/recruitment project.** It is not commissioned,
endorsed, or used by any specific school, and it contains fictional data only.

## Architecture

```
Browser
   |
Next.js Frontend  (frontend/)
   |  REST/HTTP
Go REST API        (backend/)
   |  pgx / sqlc
PostgreSQL
```

A modular monolith: one Go binary, organized into internal domain packages
(`backend/internal/*`), talking to one PostgreSQL database. No microservices,
no message queues, no Kubernetes — see `docs/decisions.md` (added in a later
milestone) for the reasoning.

## Repository layout

```
schoolhub/
├── frontend/        Next.js (App Router) + TypeScript + Tailwind CSS
├── backend/          Go REST API (Gin, pgx, sqlc — sqlc wired in M3)
│   ├── cmd/server/    main.go — entrypoint
│   ├── internal/      one package per domain (students, staff, attendance, ...)
│   ├── db/            sqlc queries + generated code (regenerate as domains grow)
│   ├── migrations/     handwritten SQL migrations (versions 000001–000004)
│   └── config/         environment-variable loading
├── docker-compose.yml  Postgres + backend for local development
└── docs/               architecture/database/API docs (added in M24)
```

## Prerequisites (local machine, not this sandbox)

- Node.js 20+ and npm
- Go 1.23+
- Docker and Docker Compose
- Git

## Local setup

```bash
git clone <your-repo-url> schoolhub
cd schoolhub
cp .env.example .env          # adjust if needed

# 1. Start Postgres + backend
docker compose up --build

# 2. In a separate terminal, start the frontend
cd frontend
cp .env.local.example .env.local
npm install
npm run dev
```

Then:

- Backend health check: http://localhost:8080/api/v1/health → `{"status":"ok","database":"connected"}`
- Frontend: http://localhost:3000

## Database migrations

Handwritten SQL migrations live in `backend/migrations/`, named
`<version>_<description>.up.sql` / `.down.sql` (golang-migrate convention).
No local `migrate` CLI install is required — a `migrate` service in
`docker-compose.yml` runs the official image against `postgres` on demand:

```bash
docker compose up -d postgres        # make sure Postgres is running first
docker compose run --rm migrate up   # apply all migrations
docker compose run --rm migrate version
docker compose run --rm migrate down 1   # roll back one migration
```

Current migrations:

| Version | Description                        |
|---------|-------------------------------------|
| 000001  | `pgcrypto` extension + shared `set_updated_at()` trigger function |
| 000002  | Identity & RBAC: `people`, `users`, `roles`, `permissions`, `role_permissions`, `user_roles` — seeded with the 5 roles and the permission codes from the spec |
| 000003  | Authentication: `sessions`, `password_reset_tokens`, `invitation_tokens` |
| 000004  | Default role → permission mapping (ADMIN: all 26 permissions; TEACHER: class-scoped set; STAFF: limited admin; PARENT/STUDENT: view-only) |

Domain-specific tables (staff, students, academics, attendance, assessments,
announcements, audit logs) are added in their own migrations in later
milestones (M6–M13, M19), not all at once.

If you'd rather install the CLI locally instead of using Docker:
[github.com/golang-migrate/migrate/tree/master/cmd/migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate),
then run `migrate -path backend/migrations -database "$DATABASE_URL" up`.

## Authentication & authorization (M4–M5)

Authentication uses opaque session tokens: the client holds a random
256-bit token in an httpOnly cookie (`schoolhub_session`); the server
stores only its SHA-256 hash. Passwords are hashed with Argon2id.
Suspended/inactive accounts are rejected at login and at every session
validation.

| Method | Path                  | Auth required |
|--------|-----------------------|---------------|
| POST   | `/api/v1/auth/login`  | no |
| POST   | `/api/v1/auth/logout` | no (harmless if already logged out) |
| GET    | `/api/v1/me`          | yes |
| GET    | `/api/v1/roles`       | yes + ADMIN role |
| GET    | `/api/v1/permissions` | yes + ADMIN role |
| GET    | `/api/v1/roles/:id/permissions` | yes + ADMIN role |
| POST   | `/api/v1/roles/:id/permissions` | yes + ADMIN role |
| DELETE | `/api/v1/roles/:id/permissions/:permission_id` | yes + ADMIN role |

`/me` and the login response include the caller's `roles` and effective
`permissions` (resolved through `user_roles` → `role_permissions`) so the
frontend can render role-aware navigation. That data is informational —
enforcement is always server-side via the `rbac.RequirePermission(code)`
middleware that domain routes (M6+) will be gated with. Assigning/revoking
permissions on roles is done through the endpoints above; the assign and
revoke operations are idempotent.

### Creating dev users (temporary seed command)

```bash
cd backend
go run ./cmd/seed -role ADMIN   -email admin@schoolhub.test   -password "ChangeMe123!"
go run ./cmd/seed -role TEACHER -email teacher@schoolhub.test -password "ChangeMe123!"
```

DEV ONLY — this stopgap exists so the auth/RBAC flow can be exercised
before the real admin-driven account creation (M6) exists, and will be
deleted once M6 provides it. These are fictional demo credentials.

## Generated database layer (sqlc)

`backend/db/queries/*.sql` holds handwritten, annotated SQL queries;
`backend/sqlc.yaml` tells [sqlc](https://sqlc.dev) to generate type-safe Go
code from those queries plus the schema in `backend/migrations/*.up.sql`,
into `backend/db/generated/`. Generated code is committed to the repo (not
gitignored) so it's reviewable without installing sqlc.

No local sqlc install needed — same pattern as migrations, via Docker:

```bash
docker compose run --rm sqlc generate
```

This needs no running database — sqlc parses the SQL files statically. Run
it again any time a query file or a migration's schema changes; the
generated code in `db/generated/` is a build artifact of those two inputs.

Current query files, one per table: `people.sql`, `users.sql`,
`roles.sql`, `permissions.sql`, `role_permissions.sql`, `user_roles.sql`,
`sessions.sql`, `password_reset_tokens.sql`, `invitation_tokens.sql`.
Notably `user_roles.sql` includes `ListPermissionsForUser` and
`UserHasPermission` — the two queries the RBAC middleware uses to enforce
authorization server-side (section 11: "Hiding a button in the frontend is
NOT security").

As each later milestone adds its own tables (M6 staff, M7 students, M8
academics, ...), it adds its own query file alongside them and re-runs
`sqlc generate` — this isn't a one-time step.

If you'd rather install sqlc locally: [docs.sqlc.dev/en/latest/overview/install.html](https://docs.sqlc.dev/en/latest/overview/install.html),
then run `sqlc generate` from `backend/`.

## Running the backend without Docker

```bash
cd backend
cp ../.env.example .env   # or set DATABASE_URL manually
go mod tidy
go run ./cmd/server
```

## Verification checklist from Milestone 0

These are the checks Milestone 0 was meant to satisfy, kept for reference.
They have since been executed successfully in a real environment (most
recently during the M5 verification pass on 2026-10-03).

```bash
# Postgres starts and is healthy
docker compose up -d postgres
docker compose ps          # postgres should show "healthy"

# Migrations apply cleanly
docker compose run --rm migrate up
docker compose run --rm migrate version   # should report 000002

# sqlc generates without errors
docker compose run --rm sqlc generate
ls backend/db/generated   # should contain .go files, not be empty

# Backend connects to Postgres and serves the health endpoint
cd backend && go mod tidy && go run ./cmd/server
curl http://localhost:8080/api/v1/health

# Frontend builds and type-checks
cd frontend && npm install && npm run typecheck && npm run build

# Frontend can reach the backend
npm run dev
# open http://localhost:3000 — confirm no console errors, page renders
```

## Environment variables

See `.env.example` at the repo root, `frontend/.env.local.example`, and
`backend/config/config.go` for what each variable does.

## Tech stack

Next.js · React · TypeScript · Tailwind CSS · Go · Gin · pgx · sqlc ·
PostgreSQL · Docker

## Roadmap

Implemented milestone-by-milestone (done: M0 init, M2 migrations, M3 sqlc,
M4 auth, M5 RBAC): init → migrations → sqlc → auth → RBAC →
people/staff → students/guardians → academics → enrollment → teacher
assignments → attendance → assessments/results → announcements → dashboards
(admin/teacher/parent/student) → reports → audit logs → security hardening →
testing → seed data → deployment → documentation.
