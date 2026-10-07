# SchoolHub — Project Roadmap & Progress Log

This file is the working handoff note for the project. It is meant to make the
status of the application obvious to anyone picking up the repo, without having
to re-read the whole codebase or guess what is missing.

The project follows the roadmap defined in the milestone plan from M0 to M24.

## Current status

- Last completed milestone: M16
- Current next milestone: M17
- Current branch/project state: M16 (Parent portal) is implemented, unit tested, and wired into the API router and Next.js frontend. Backend build, full test suite (12 packages), and frontend production build all verified green in the latest session. The project is ready for M17 (Student portal).
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
| M10 | Teacher assignments | Done | Teacher → class → subject assignments, audit logging; fixed gin wildcard conflict |
| M11 | Attendance | Done | Attendance sessions, register capture, mark corrections, student/class summaries, audit logging |
| M12 | Assessments and results | Done | Assessments header CRUD, score sheet entry, single result corrections, student results history, audit logging |
| M13 | Announcements | Done | School-wide broadcasts, role/class-targeted communication, draft/published/archived status flows, audience feed filtering, audit logging |
| M14 | Admin dashboard | Done | Institutional KPIs, active academic calendar context, daily attendance rate analytics, recent announcements, live audit feed, administrative quick actions, Next.js admin page |
| M15 | Teacher portal | Done | Teacher overview, assigned classes/subjects, class rosters with guardian contacts, recent attendance/assessment activity, announcements; Next.js teacher page |
| M16 | Parent portal | Done | Guardian overview, linked children with attendance/results aggregates, per-child detail; row-level scoping via student_guardians; Next.js parent page |
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
- Migration 000010: `teacher_assignments` table (teacher staff record → class → subject), unique per teacher/class/subject, cascading FKs
- sqlc query layer: create, get, filtered/paginated list/count, delete, `HasClassSubject`
- `internal/assignments` service: full chain validation (subject must be in the class curriculum via class_subjects) and transactional audit logging
- Handlers and routes: reads under `class.view`, mutations under `class.update`; flat and `/teachers/:id/assignments` nested forms
- Handler unit tests and live end-to-end verification against the running stack
- Fixed a startup-breaking gin wildcard conflict in the M9 nested enrollment routes (`:student_id` → `:id`, path param now honored)

### M11 — Attendance
- Migration 000011: `attendance_sessions` (one register event per class/subject/term/date) and `attendance_records` (one mark per student per session, status CHECK constraint for PRESENT/ABSENT/LATE/EXCUSED)
- sqlc query layer: session create, get with joins, paginated filtered list/count; record create, get detail with joins, single mark update; student and class per-status aggregate summaries
- `internal/attendance` service: full chain validation (subject in class curriculum, term in class academic year, date in term window, active enrollment verification for each marked student) and transactional audit logging (`ATTENDANCE_RECORDED`, `ATTENDANCE_RECORD_UPDATED`)
- HTTP handlers and route wiring in `cmd/server/main.go`:
  - `attendance.view`: `GET /attendance/sessions`, `GET /attendance/sessions/:id`, `GET /students/:id/attendance/summary`, `GET /classes/:id/attendance/summary`
  - `attendance.record`: `POST /attendance/sessions`
  - `attendance.update`: `PATCH /attendance/records/:id`
- Focused unit tests covering all attendance endpoints and validation/error branches

### M12 — Assessments and results
- Migration 000012: `assessments` (graded event header per class/subject/term/title with type CHECK, max_score, weight, assessment_date) and `results` (per-student score, grade, remarks)
- sqlc query layer: assessment create, get with joins, paginated filtered list/count, update, delete; results score sheet create, assessment score sheet list, single result get detail with joins, single result update, student results history across assessments
- `internal/assessments` service: chain validation (subject in class curriculum, term in class academic year, date in term window, active enrollment check per student, exact 0 <= score <= max_score boundary validation) and transactional audit logging (`ASSESSMENT_CREATED`, `ASSESSMENT_UPDATED`, `ASSESSMENT_DELETED`, `RESULTS_RECORDED`, `RESULT_UPDATED`)
- HTTP handlers and route wiring in `cmd/server/main.go`:
  - `assessment.view`: `GET /assessments`, `GET /assessments/:id`
  - `assessment.create`: `POST /assessments`
  - `assessment.update`: `PATCH /assessments/:id`, `DELETE /assessments/:id`
  - `result.view`: `GET /assessments/:id/results`, `GET /results/:id`, `GET /students/:id/results`
  - `result.create`: `POST /assessments/:id/results`
  - `result.update`: `PATCH /results/:id`
- Focused unit tests covering all assessment and result endpoints and validation/conflict error paths

### M13 — Announcements
- Migration 000013: `announcements` table with target integrity CHECK constraint (`ALL`, `ROLE`, `CLASS`), priority (`LOW`, `NORMAL`, `HIGH`, `URGENT`), draft/published/archived status flow, published/expires timestamps, and cascading FKs
- sqlc query layer: announcement create, get by ID with joined author and target class, paginated list/count with search and status/target/role/class/author filters, paginated audience feed query resolving role/class membership, status update, field updates, and deletion
- `internal/announcements` service: target integrity validation (role validation against system roles, class existence check), author/admin ownership enforcement for update/publish/archive/delete, and transactional audit logging (`ANNOUNCEMENT_CREATED`, `ANNOUNCEMENT_UPDATED`, `ANNOUNCEMENT_PUBLISHED`, `ANNOUNCEMENT_ARCHIVED`, `ANNOUNCEMENT_DELETED`)
- HTTP handlers and route wiring in `cmd/server/main.go`:
  - `announcement.view`: `GET /announcements` (feed-aware for regular users / admin filtered query), `GET /announcements/:id`
  - `announcement.create`: `POST /announcements`
  - `announcement.update`: `PATCH /announcements/:id`, `POST /announcements/:id/publish`, `POST /announcements/:id/archive`, `DELETE /announcements/:id`
- Focused unit tests covering audience feeds, admin filters, get, create validation, update/publish, and delete authorization
- Live end-to-end verification against the running docker backend: draft creation, get by ID, patch update, publishing, teacher feed visibility, archiving, deletion, and verified transactional audit log records

### M14 — Admin dashboard
- Aggregated sqlc query layer: institutional counts (active students, active staff, active teachers, total classes, and active enrollments); active academic year and term context; daily attendance session and overall present rate calculations; top recent announcements with joined author and class metadata; recent audit log records with actor identity and JSON metadata.
- `internal/dashboard` service: data aggregation combining counts, academic calendar context, daily attendance metrics, recent announcements, activity feed, and categorized administrative quick actions (`STUDENTS`, `STAFF`, `ACADEMICS`, `COMMUNICATION`, `ADMIN`).
- HTTP handlers & route wiring in `cmd/server/main.go`:
  - `GET /api/v1/dashboard/admin` gated on the `ADMIN` role with `RequireRole(rbacService, "ADMIN")`.
- Unit tests covering successful KPI aggregation, data decoding, and database failure handling.
- Frontend Next.js integration:
  - TypeScript models in `frontend/types/dashboard.ts`
  - Service layer in `frontend/services/dashboard.ts`
  - Admin dashboard page in `frontend/app/dashboard/admin/page.tsx` featuring responsive KPI summary cards, present/late/absent attendance breakdown, categorized quick action launchers, announcements feed, and audit timeline
  - Production build verified with `npm run build`.
- Live end-to-end verification against the running docker backend: authenticated admin access (`200 OK`), unauthenticated rejection (`401 Unauthorized`), non-admin rejection (`403 Forbidden`).

### M15 — Teacher portal
- sqlc query layer (`db/queries/teacher_portal.sql` + generated code): teacher profile and active academic year/term context, assignment counts, assigned classes with subject/enrollment metadata, recent attendance sessions, recent assessments, teacher-targeted announcements, class roster with guardian contacts, and a `HasClassSubject` assignment check.
- `internal/teacherportal` service: overview aggregation (profile, academic context, counts, assigned classes, recent attendance registers, recent assessments, announcements), assigned-class list, and class roster gated on the assignment check so a teacher cannot read another class's roster.
- HTTP handlers and route wiring in `cmd/server/main.go`, all read-only and gated on the `TEACHER` role:
  - `GET /portal/teacher/overview`
  - `GET /portal/teacher/classes`
  - `GET /portal/teacher/classes/:id/roster`
- Handler unit tests covering overview success/failure, class list, roster for an assigned class, roster rejection for an unassigned class, and invalid UUID handling.
- Frontend Next.js integration: `frontend/types/teacher_portal.ts`, `frontend/services/teacher_portal.ts`, and page `frontend/app/portal/teacher/page.tsx`; production build verified with `npm run build`.
- Verification: backend build, focused unit tests, and frontend production build all green (live end-to-end verification against the running stack not redone in the latest session).

### M16 — Parent portal
- sqlc query layer (`db/queries/parent_portal.sql` + generated code): guardian profile, linked children with active enrollment and lifetime attendance/results aggregates, per-child identity/attendance/results, and a guardian-student link check via `student_guardians`.
- `internal/parentportal` service: overview (profile, targeted announcements, children) and per-child detail (identity, attendance marks, full results history). Row-level scoping — a parent sees only their own children — is enforced in the service via `student_guardians` (RBAC cannot express row ownership); an unlinked child request resolves to 404.
- HTTP handlers and route wiring in `cmd/server/main.go`, all read-only and gated on the `PARENT` role:
  - `GET /portal/parent/overview`
  - `GET /portal/parent/children/:id`
- Handler unit tests covering overview success/failure, linked-child detail, unlinked-child rejection (403 Forbidden — row-ownership violation; 404 is reserved for a PARENT account with no guardians row), invalid UUID (400), and missing child (404).
- Frontend Next.js integration: `frontend/types/parent_portal.ts`, `frontend/services/parent_portal.ts`, and page `frontend/app/portal/parent/page.tsx`.
- Verification: backend build, full `go test ./...` suite (12 packages passing), and frontend `npm run build` all green.

---

## Current milestone: M17 summary

M17 is the next milestone. It introduces the Student Portal for student-facing
personal data. The expected work, following the same read-only self-scoped
pattern as M15/M16:

- student profile and personal data view
- enrollment history and current class/academic context
- own attendance and results summaries
- student-targeted announcements

Operational note: another project's container (`lightex-db`) occupies host port 5432 in this environment. Run the schoolhub stack with `POSTGRES_PORT=5433 docker compose up -d` and point the local server at `postgres://schoolhub:schoolhub@localhost:5433/schoolhub?sslmode=disable`. The containerized backend is unaffected (internal network).

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
- Most recent completed domain milestones: M14 admin dashboard, M15 teacher portal, M16 parent portal
- Current milestone to implement: M17 student portal

## Recommended continuation pattern

When starting the next milestone, use this sequence:

1. Add/modify migration files
2. Add SQL queries
3. Regenerate sqlc output
4. Implement service logic
5. Add handlers and route wiring
6. Test the endpoints and business flows
7. Update this file with the new milestone status
