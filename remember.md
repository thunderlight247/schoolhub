# SchoolHub — Project Roadmap & Progress Log

This file is the working handoff note for the project. It is meant to make the
status of the application obvious to anyone picking up the repo, without having
to re-read the whole codebase or guess what is missing.

The project follows the roadmap defined in the milestone plan from M0 to M24.

## Current status

- Last completed milestone: M10
- Current next milestone: M11
- Current branch/project state: M10 (Teacher assignments) has been fully implemented, migrated, unit tested, and verified end-to-end against the running stack. The project is ready for M11 (Attendance).
- Project direction: continue in milestone order, one milestone at a time

---

## Milestone roadmap

| Milestone | Title | Status | Notes |
|---|---|---|---|
| M0 | Project initialization | Done | Docker, repo structure, basic frontend/backend, health check, env config |
| M1 | Architecture/folder structure | Done | Modular monolith layout and domain boundaries |
| M2 | PostgreSQL migrations | Done | Identity and RBAC core schema |
| M3 | sqlc configuration + generated DB layer | Done | Query layer and generated Go code |
| M4 | Authentication | Done | Login, logout, session auth, password hashing |
| M5 | RBAC/permissions | Done | Role and permission enforcement |
| M6 | People and staff | Done | Staff domain, departments, positions, accounts, audit logging |
| M7 | Students and guardians | Done | Students, guardians, relationships, student/guardian access, unit tested and wired |
| M8 | Academic structure | Done | Academic years, terms, programmes, subjects, classes, class-subject assignments |
| M9 | Enrollment | Done | Enrollment history, class membership, status transitions, audit logging |
| M10 | Teacher assignments | Done | Teacher → class → subject assignments, audit logging; also fixed a gin wildcard conflict in M9 nested enrollment routes |
| M11 | Attendance | Pending | Attendance capture and reporting |
| M12 | Assessments and results | Pending | Assessments, scores, grades, remarks |
| M13 | Announcements | Pending | Broadcasts and targeted communication |
| M14 | Admin dashboard | Pending | KPIs and quick actions |
| M15 | Teacher portal | Pending | Teacher-specific views and workflows |
| M16 | Parent portal | Pending | Child data, attendance, results, announcements |
| M17 | Student portal | Pending | Student profile and personal data |
| M18 | Reports | Pending | Summary and analytical outputs |
| M19 | Audit logs | Pending | Authorized admin audit review |
| M20 | Security hardening | Pending | protections and production readiness |
| M21 | Testing | Pending | backend/front-end validation and flows |
| M22 | Seed/demo environment | Pending | realistic demo users and data |
| M23 | Deployment | Pending | Vercel + Go hosting + Postgres |
| M24 | Documentation & portfolio prep | Pending | README, docs, final polish |

---

## What has been completed so far

### M0 — Project initialization
- Project repository structure created
- Next.js frontend initialized
- Go backend initialized
- Docker Compose configured
- PostgreSQL local development setup prepared
- environment-variable handling configured
- minimal health-check endpoint implemented
- minimal frontend page implemented

### M1 — Architecture
- modular monolith direction established
- backend organized around internal domains
- API entrypoint and service layering pattern recognized and followed

### M2 — PostgreSQL migrations
- migration foundation for identity and RBAC established
- people, users, roles, permissions, and role join tables created
- default role/value structure created for the core system

### M3 — sqlc setup
- SQL query files added
- generated DB models and access layer created
- project uses sqlc + Go typed queries as part of the backend architecture

### M4 — Authentication
- login and logout flow created
- secure session/cookie handling implemented
- password hashing support added
- protected endpoints established
- account/session validation pattern in place

### M5 — RBAC and permissions
- permission checks enforced server-side
- role-permission mapping implemented
- admin-only management routes created
- role-aware authorization flow established

### M6 — People and staff
- staff domain implemented
- departments and staff positions created
- staff records and assignment logic implemented
- account creation and role assignment for staff added
- employment status updates added
- audit logging for mutation flows implemented
- public-facing staff routes wired into the API

### M7 — Students and guardians
- student and guardian domain implemented
- student profile records and relationships created
- guardian mappings and access rules established
- student-facing and parent-facing data boundaries defined
- routes and permissions for student/guardian access wired into API router
- unit tests added for student and guardian handlers

### M8 — Academic structure
- Migration 000008: `academic_years`, `terms`, `programmes`, `subjects`, `classes`, `class_subjects`
- Unique current flag constraint on academic_years and terms
- sqlc queries for all academic domain entities
- `internal/academics` service logic with transactional audit logging (`audit.Action*` for M8)
- HTTP handlers for full academic structure CRUD and class-subject mappings
- Gated endpoints wired under `class.view`, `class.create`, and `class.update` permissions
- Comprehensive unit tests covering handlers and error handling

### M9 — Enrollment
- Migration 000009: `enrollments` table with unique active-enrollment-per-year protection
- sqlc query layer for create/list/get/update-status operations
- `internal/enrollment` service with validation and audit logging
- HTTP handlers and routes wired into `cmd/server/main.go`
- Focused unit tests covering the enrollment API flow

### M10 — Teacher assignments
- Migration 000010: `teacher_assignments` table (teacher staff record → class →
  subject), unique per teacher/class/subject, cascading FKs
- sqlc query layer: create, get, filtered/paginated list/count, delete, `HasClassSubject`
- `internal/assignments` service: full chain validation (subject must be in the
  class curriculum via class_subjects) and transactional audit logging
- Handlers and routes: reads under `class.view`, mutations under `class.update`;
  flat and `/teachers/:id/assignments` nested forms
- Handler unit tests and live end-to-end verification against the running stack
- Fixed a startup-breaking gin wildcard conflict in the M9 nested enrollment routes
  (`:student_id` → `:id`, path param now honored)

---

## Current milestone: M11 summary

M11 is the next milestone. It introduces attendance as the next domain
boundary. The expected work:

- attendance capture per class, subject, and term (teacher-facing)
- attendance records tied to enrolled students and their classes
- attendance status values and update flows, permission-gated
- attendance summaries/reports groundwork for the parent/student portals
- audit rows for attendance mutations

The most recent completed milestone was M10 teacher assignments, which included:
- migration 000010: `teacher_assignments` table (teacher staff record → class →
  subject, unique per teacher/class/subject, cascading FKs)
- sqlc query layer: create, get (joined detail row), list/count with filters
  (teacher/class/subject/academic year/search), delete, and `HasClassSubject`
- `internal/assignments` service with full chain validation (staff, class,
  subject must exist; subject must be in the class curriculum) and
  transactional audit logging (`TEACHER_ASSIGNMENT_CREATED`,
  `TEACHER_ASSIGNMENT_REMOVED`)
- HTTP handlers and routes in `cmd/server/main.go`: reads under `class.view`,
  create/remove under `class.update`; flat `/teacher-assignments` plus nested
  `/teachers/:id/assignments` (path teacher wins over body/query)
- focused handler unit tests (stubbed service, all endpoints + error paths)
- end-to-end verification against the live stack: created year/subject/class/
  class-subject through the real API, exercised 201/409/400/200/404 flows,
  confirmed audit rows, cleaned up
- fix carried in: M9's nested enrollment routes used `:student_id` while
  `/students/:id` already existed — gin panics at registration on conflicting
  wildcard names, so the server could not boot. Routes renamed to `:id` and
  the enrollment handlers now honor the path param (path wins over query/body)

Operational note discovered during M10: another project's container
(`lightex-db`) occupies host port 5432 in this environment. Run the schoolhub
stack with `POSTGRES_PORT=5433 docker compose up -d` and point the local
server at `postgres://schoolhub:schoolhub@localhost:5433/schoolhub?sslmode=disable`.
The containerized backend is unaffected (internal network).

---

## Working rules for future milestones

- Implement one milestone at a time.
- Keep discipline: migration, sqlc, service, handlers, routes, tests.
- Do not jump ahead into later domains.
- The project remains a modular monolith with clear domain boundaries.
- Any change to architecture must be explained before it is adopted.

---

## Quick operational notes

- Main backend: backend/
- Main frontend: frontend/
- DB layer: backend/db/
- Migrations: backend/migrations/
- App entrypoint: backend/cmd/server/main.go
- Most recent completed domain milestone: M10 teacher assignments
- Current milestone to implement: M11 attendance

## Recommended continuation pattern

When starting the next milestone, use this sequence:

1. Add/modify migration files
2. Add SQL queries
3. Regenerate sqlc output
4. Implement service logic
5. Add handlers and route wiring
6. Test the endpoints and business flows
7. Update this file with the new milestone status
