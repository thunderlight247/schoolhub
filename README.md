# SchoolHub

A full-stack school administration & management portal, built as a portfolio
prototype demonstrating practical software engineering: Next.js/TypeScript on
the frontend, Go on the backend, PostgreSQL for storage.

> **Status:** Milestone 0 — project initialization. Authentication, RBAC, and
> the domain model (students, staff, attendance, results, etc.) are not yet
> implemented; they arrive in later milestones.

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
│   ├── db/            sqlc queries + generated code (empty until M3)
│   ├── migrations/     handwritten SQL migrations (empty until M2)
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

Domain-specific tables (staff, students, academics, attendance, assessments,
announcements, audit logs) are added in their own migrations in later
milestones (M6–M13, M19), not all at once.

If you'd rather install the CLI locally instead of using Docker:
[github.com/golang-migrate/migrate/tree/master/cmd/migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate),
then run `migrate -path backend/migrations -database "$DATABASE_URL" up`.

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

Current query files, one per M2 table: `people.sql`, `users.sql`,
`roles.sql`, `permissions.sql`, `role_permissions.sql`, `user_roles.sql`.
Notably `user_roles.sql` includes `ListPermissionsForUser` and
`UserHasPermission` — the two queries the auth middleware (M4/M5) will use
to enforce authorization server-side (section 11: "Hiding a button in the
frontend is NOT security").

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

## Verification checklist for this milestone

These are the checks Milestone 0 is meant to satisfy. Run them locally —
**they have not been executed in the sandbox this scaffold was generated in**,
which has no Go/Docker toolchain and no network access. See "Known issues"
below.

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

Implemented milestone-by-milestone: init → migrations → sqlc → auth → RBAC →
people/staff → students/guardians → academics → enrollment → teacher
assignments → attendance → assessments/results → announcements → dashboards
(admin/teacher/parent/student) → reports → audit logs → security hardening →
testing → seed data → deployment → documentation.
