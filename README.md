# SchoolHub

A full-stack school administration & management portal built as a portfolio
project that demonstrates practical engineering: Next.js + TypeScript on the
frontend, Go on the backend, and PostgreSQL for persistence.

> Status: Milestone 8 — Academic structure is implemented and ready for
> verification. The project now includes the identity/RBAC foundation,
> auth, the M6 staff domain, M7 students/guardians domain, and M8 academic structure.

**This is a personal portfolio/recruitment project.** It is not commissioned,
endorsed, or used by any specific school, and it contains fictional data only.

## Current project state

The repository is now at the end of M8. The codebase includes:

- project scaffolding and local container setup
- migrations from 000001 through 000008
- sqlc-generated database access code
- authentication and session management
- admin RBAC and permission enforcement
- staff management, reference data, and audit log writes for mutations
- students and guardians: registration, search/filter, edit, archive, and
  guardian linking, all audit-logged
- academic structure: academic years, terms, programmes, subjects, classes,
  and class-subject assignments

The next planned milestone is M9: enrollment.

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

This is a modular monolith: one Go binary, organized into domain packages under
`backend/internal/*`, backed by a single PostgreSQL database.

## Repository layout

```
schoolhub/
├── frontend/         Next.js (App Router) + TypeScript + Tailwind CSS
├── backend/          Go REST API (Gin, pgx, sqlc)
│   ├── cmd/server/   main.go — API entrypoint
│   ├── internal/     domain packages: auth, rbac, staff, students, audit, etc.
│   ├── db/           sqlc queries and generated Go models
│   ├── migrations/   SQL migrations (000001–000007)
│   ├── config/      environment/config loading
│   └── cmd/seed      temporary demo user seeding
├── docker-compose.yml
├── docs/
├── README.md
├── remember.md
└── .env.example
```

Note: `internal/guardians/` (stubbed at project init) is removed as of M7 —
guardians are implemented inside `internal/students` instead. See migration
000007's comment for why (same reasoning M6 used to fold departments and
staff positions into `internal/staff`).

## Prerequisites

- Node.js 20+
- Go 1.22+
- Docker and Docker Compose
- Git

## Local setup

```bash
git clone <your-repo-url> schoolhub
cd schoolhub
cp .env.example .env

# Start core services
docker compose up --build

# Start the frontend in a second terminal
cd frontend
cp .env.local.example .env.local
npm install
npm run dev
```

Then:

- Backend health: http://localhost:8080/api/v1/health
- Frontend: http://localhost:3000

## Migrations and generated code

The app uses handwritten SQL migrations in `backend/migrations/` and sqlc for
Go models and query wrappers under `backend/db/generated/`.

Current migration stack:

- 000001: pgcrypto + shared DB helper trigger
- 000002: identity and RBAC tables
- 000003: sessions and token tables
- 000004: default role-to-permission mappings
- 000005: departments, staff positions, and staff
- 000006: audit logs
- 000007: students, guardians, and student_guardians
- 000008: academic structure (academic_years, terms, programmes, subjects, classes, class_subjects)
- 000009: enrollment history
- 000010: teacher assignments

To apply them:

```bash
docker compose up -d postgres
docker compose run --rm migrate up
docker compose run --rm migrate version
```

If you change SQL or schema files, regenerate database bindings:

```bash
docker compose run --rm sqlc generate
```

## Authentication and authorization

The backend currently supports:

- login/logout with the `schoolhub_session` cookie
- session validation and auth middleware
- role-based access checks
- permission-gated business routes

Demo users for local development:

```bash
cd backend
go run ./cmd/seed -role ADMIN -email admin@schoolhub.test -password "ChangeMe123!"
go run ./cmd/seed -role TEACHER -email teacher@schoolhub.test -password "ChangeMe123!"
```

These are fictional dev credentials and are meant for local testing only.

## Current M6 feature set

The M6 work includes:

- departments and staff positions reference data
- staff creation and lookup
- staff assignment metadata and employment status updates
- account creation for staff members and role assignment
- audit rows written as part of each mutation transaction
- permission-gated staff routes in `cmd/server/main.go`

## Current M7 feature set

The M7 work includes:

- student registration, search/filter, profile view, person-field edit,
  and archive/reactivate (section 14)
- guardian registration, search, profile view, and person-field/occupation
  edit, standalone from any particular student (section 15)
- linking and unlinking an existing guardian to/from a student, with a
  free-text relationship label (a student may have multiple guardians —
  "one student = one parent" is not assumed)
- audit rows written as part of each mutation transaction, same pattern as
  M6: `STUDENT_CREATED`, `STUDENT_UPDATED`, `STUDENT_STATUS_CHANGED`,
  `GUARDIAN_CREATED`, `GUARDIAN_UPDATED`, `STUDENT_GUARDIAN_LINKED`,
  `STUDENT_GUARDIAN_UNLINKED`
- permission-gated student/guardian routes in `cmd/server/main.go`, reusing
  the four `student.*` permission codes from section 11 for guardians too
  (section 11 has no separate `guardian.*` codes)

Not in M7, deliberately: no class/enrollment link on students (that's
M9's `enrollments` table — section 14 explicitly says not to put a
permanent class field on the student record), and no guardian login/portal
## Current M8 feature set

The M8 work includes:

- academic years: institutional academic calendars, date bounds validation,
  and active year toggling (`ACADEMIC_YEAR_CREATED`, `ACADEMIC_YEAR_UPDATED`,
  `ACADEMIC_YEAR_SET_CURRENT`)
- terms/semesters: academic subdivisions scoped to academic years, with
  unique term names per year and active term management (`TERM_CREATED`,
  `TERM_UPDATED`, `TERM_SET_CURRENT`)
- programmes: academic curriculum streams/tracks with unique codes and names
  (`PROGRAMME_CREATED`, `PROGRAMME_UPDATED`)
- subjects: institutional subject catalog categorized by Core or Elective,
  optionally mapped to departments (`SUBJECT_CREATED`, `SUBJECT_UPDATED`)
- classes: student cohort classrooms tied to an academic year and programme,
  with capacity controls and uniqueness guarantees (`CLASS_CREATED`, `CLASS_UPDATED`)
- class-subject assignments: curriculum mapping specifying which subjects are
  taught in which class (`CLASS_SUBJECT_ASSIGNED`, `CLASS_SUBJECT_REMOVED`)
- full audit logging across all mutations executed inside atomic transactions
- permission gating wired to `class.view`, `class.create`, and `class.update`

Key API areas:

- `/api/v1/staff`
- `/api/v1/departments`
- `/api/v1/staff-positions`
- `/api/v1/students`
- `/api/v1/students/:id/guardians`
- `/api/v1/guardians`
- `/api/v1/academic-years`
- `/api/v1/academic-years/current`
- `/api/v1/terms`
- `/api/v1/terms/current`
- `/api/v1/programmes`
- `/api/v1/subjects`
- `/api/v1/classes`
- `/api/v1/classes/:id/subjects`
- `/api/v1/enrollments`
- `/api/v1/teacher-assignments`
- `/api/v1/auth/*`
- `/api/v1/roles` and `/api/v1/permissions` (admin-only)

## Current M9 feature set

The M9 work includes:

- enrollment history records that separate student identity from class membership
- enrollment creation tied to a student, academic year, and class
- duplication protection for active enrollments in the same academic year
- status transitions for ENROLLED, COMPLETED, TRANSFERRED, and WITHDRAWN
- audit rows written as part of each enrollment mutation transaction
- permission-gated enrollment routes under the existing class-view/create/update checks

Key API areas:

- `/api/v1/enrollments`
- `/api/v1/students/:id/enrollments`
- `/api/v1/enrollments/:id`
- `/api/v1/enrollments/:id/status`

## Current M10 feature set

The M10 work includes:

- teacher assignment records tying a teacher (staff record), a class, and
  a subject together for teaching coverage
- creation validation across the whole chain: the staff record, class, and
  subject must all exist, and the subject must already be part of the
  class's curriculum (class_subjects, M8) — an assignment covers teaching,
  it never widens the curriculum
- duplicate protection: one assignment per teacher per class-subject
  pairing (co-teaching stays possible — a different teacher may hold the
  same class-subject)
- filtered/paginated listing (by teacher, class, subject, academic year,
  or free-text search) and assignment removal with metadata-preserving
  audit rows (`TEACHER_ASSIGNMENT_CREATED`, `TEACHER_ASSIGNMENT_REMOVED`)
- permission-gated routes: reads under `class.view`, create/remove under
  `class.update`, mirroring the M8 class-subject mappings this domain
  sits next to

Key API areas:

- `/api/v1/teacher-assignments`
- `/api/v1/teacher-assignments/:id`
- `/api/v1/teachers/:id/assignments`

## How to continue from here

1. Make sure the database and backend are running.
2. Apply migrations if needed (`docker compose run --rm migrate up`).
3. Start the API and the frontend.
4. Log in as the seeded admin and exercise the staff, student, academic, and enrollment workflows.
5. Move to the next milestone: M11 attendance.

Example flow:

```bash
cd /media/light/Grace/schoolhub_1
docker compose up -d postgres
docker compose run --rm migrate up
cd backend && go run ./cmd/server

# in another terminal
cd frontend && npm run dev
```

Then open the app and log in with:

- Email: admin@schoolhub.test
- Password: ChangeMe123!

## Verification notes

The backend and all domain packages build and test successfully:

```bash
cd backend && go build ./... && go test ./...
cd frontend && npm run typecheck
```

## Roadmap

Completed:

- M0 project setup
- M1 architecture
- M2 migrations
- M3 sqlc
- M4 auth
- M5 RBAC
- M6 people and staff
- M7 students and guardians
- M8 academic structure
- M9 enrollment
- M10 teacher assignments

Next:

- M11 attendance
- M12 assessments and results
- M13 announcements
- M14 admin dashboard
- M15 teacher portal
- M16 parent portal
- M17 student portal
- M18 reports
- M19 audit logs
- M20 security hardening
- M21 testing
- M22 seed/demo environment
- M23 deployment
- M24 documentation and portfolio preparation

## Tech stack

Next.js · TypeScript · Tailwind CSS · Go · Gin · pgx · sqlc · PostgreSQL · Docker

