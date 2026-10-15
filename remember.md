# SchoolHub — Project Roadmap & Progress Log

This file is the working handoff note for the project. It is meant to make the
status of the application obvious to anyone picking up the repo, without having
to re-read the whole codebase or guess what is missing.

The project follows the roadmap defined in the milestone plan from M0 to M24,
but the repo itself is already ahead of the original linear doc sequence in the
backend. The authoritative state is the codebase and tests, not a stale roadmap
assumption.

## Current status

- Last completed milestone: Access-control layer (master admin + blocking), after the expert pass
- Current active milestone: M30
- Current branch/project state: **M30 DONE** — announcements composer UI (`/announcements`, `frontend/scripts/announcements-check.js` 15/15). Before that: account-access control (**MASTER_ADMIN** role, migration 000014; only master admins may revoke other admins — `admin.manage` + MASTER_ADMIN double-gated), any admin can **block/unblock** teacher/staff/parent/student accounts mid-session (blocking = `users.status → SUSPENDED`; ValidateSession re-checks status on every request so lockout is instant), `/access` UI, Clerk-native invitations on provisioning (best-effort `backend/internal/auth/clerk_invite.go`), dedicated **"Create parent access"** button on `/guardians/[id]`, `parent-access-check.js` 14/14, and `block-parent-browser-check.js` **11/11** real-browser proof of mid-session blocking. Next milestone: M31.
- Frontend scope: local demo and validation only; not a milestone acceptance requirement unless a milestone explicitly includes UI work.
- Project direction: continue in backend/domain order, one milestone at a time

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
| M17 | Student portal | Done | Self-scoped student profile, academic context, attendance register, grades history, enrollment timeline, announcements; Next.js student page |
| M18 | Reports | Done | Enrollment, class attendance, class performance, and student report cards with `report.view` gating |
| M19 | Audit logs | Done | Authorized admin audit review with `audit.view` gating |
| M20 | Security hardening | Done | login rate limiting and least-privilege app DB credentials |
| M21 | Testing | Done | Full backend suite green, frontend build green, live E2E pass over auth/RBAC/portals/reports/audit/rate-limiting; environment fixes documented |
| M22 | Seed/demo environment | Done | `backend/cmd/demo-seed`: idempotent demo dataset (fixed UUIDs + ON CONFLICT DO NOTHING); 5 demo accounts with `ChangeMe123!`; README section added |
| M23 | Frontend design system & admin shell | Done | Login page + AuthProvider + ProtectedRoute + AppShell (sidebar/topbar/sign-out) adopted by all 5 screens; UI kit in `frontend/components/ui/` (DataTable, SearchInput, Pagination, Modal, form fields); browser-verified via `frontend/scripts/m23-browser-check.js` |
| M24 | Admin: staff & reference data UI | Done | `/staff` directory (search + department/status/category filters + pagination + create modal), `/staff/[id]` profile (personal edit, assignment full-replace, status control, account create + role edit), `/staff/setup` (departments/positions). Fixed backend CORS PUT gap; browser-verified via `frontend/scripts/m24-browser-check.js` (10/10) |
| M25 | Admin: students & guardians UI | Done | `/students` (stat cards + search + status filter + pagination + enroll modal), `/students/[id]` (edit, archive/re-activate, guardian link/unlink with confirmation), `/guardians` + `/guardians/[id]` (directory, profile with linked children). UI kit upgraded to the reference bar. Browser-verified via `frontend/scripts/m25-browser-check.js` (13/13) |
| M26 | Admin: academic structure UI | Done | Tabbed `/academics`: years/terms (current-flag move + edit), programmes, subjects, classes, class curriculum assign/remove. Browser-verified via `frontend/scripts/m26-browser-check.js` (10/10) |
| M27 | Admin: enrollment & teacher assignment UI | Done | `/enrollments` register (status/class/year pill filters, search, pagination, per-row status transition select, search-pick enroll modal with year→class dependency), `/teaching` assignments directory (teacher/class/subject columns, curriculum-accurate create modal, remove action), student-profile enrollment history. Backend gained `student_number` on the enrollment list DTO. Browser-verified via `frontend/scripts/m27-browser-check.js` (13/13) |
| M28 | Attendance UI | Done | Teacher register-taking screen, correction flow, class summary, duplicate conflict rejection; admin school-wide attendance view. Covers M11's backend. Browser-verified via `frontend/scripts/m28-browser-check.js` (21/21). |
| M29 | Assessments & results UI | Done | Admin/teacher assessment creation, score-sheet entry grid, result corrections. (Student/parent-facing views already exist from M16/M17.) Covers M12's backend. Browser-verified via `frontend/scripts/m29-browser-check.js`. |
| EXP | Professionalization pass | Done | Real school landing page (SchoolHub Academy, public images), demo text removed, optional Clerk hybrid-bridge auth, optional Resend email (welcome + announcement notifications). Verified `expert-pass-check.js` 12/12. |
| ACC | Access-control layer | Done | MASTER_ADMIN role above ADMIN (only role that can add/remove other admins via `admin.manage`), all admins can block/unblock teacher/staff/parent/student accounts mid-session (`account.block`), `/access` UI. Verified `access-control-check.js` 15/15 + `block-parent-browser-check.js` 11/11 + go test/vet + tsc/build. |
| M30 | Announcements composer UI | Done | `/announcements` page: composer modal (draft/edit, ALL/ROLE/CLASS targeting, priority, expiry), directory with status/audience/search filters + pagination, Publish/Archive/Delete per row (school-wide publish triggers backend's staff email). Verified `announcements-check.js` 15/15 + tsc/build. |
| M31 | Reports UI | Done | `/reports` Reports Center: four tabs over M18's endpoints — Enrollment Summary (KPI + per-class utilization/gender table), Class Attendance Analytics (class/term/date filters, per-student rates), Class Academic Performance (assessment averages), Student Report Card (printable). Verified `reports-check.js` 15/15 + tsc/build. |
| M32 | Audit log viewer UI | Done | `/audit` admin screen over M19's endpoints — filterable paginated log with action/resource/date filters, detail modal with decoded JSONB metadata, teacher/master-admin nav. Verified `audit-check.js` 18/18 + tsc/build. |
| M33 | Frontend polish & responsive pass | Done | Accessible Tabs (role=tablist/aria-selected/arrow-key, `components/ui/tabs.tsx`) refactoring the 4 tabbed pages; modal focus-in/restore; form fields auto-id `<label htmlFor>` + `aria-invalid`; AppShell landmarks (aria-current, nav labels) + wrappable topbar (fixed /staff 405px mobile overflow); verified `polish-check.js` **41/41** (16-page desktop+390px overflow sweep, tabs a11y on /reports + student portal, modal focus/Escape on /audit, composer label htmlFor integrity). tsc/build green. |
| M34 | Master-admin & real credentials | Done | User-requested (Oct 13 2026): (1) fix the NetworkError/dev-overlay hit when an admin signs out and logs in as teacher/parent (refresh fixes it — unprofessional); (2) master-admin add-admin UI (control surface for `admin.manage` — only MASTER_ADMIN invites other admins; ordinary admins invite teacher/parent only; teachers invite parents; every inviter can see invitees and block account; MASTER_ADMIN can also block other admins); (3) remove demo emails/passwords from DB and UI — new master admin account is **thunderlight2477@gmail.com** with initial password **John2477@** which MUST trigger a change-password prompt on first login, so even the developer can't log in afterwards. (1) resolved by the M34 sign-out hard redirect (window.location.replace) shared with the gate; (2)+(3) shipped — see the M34 section below. |
| M35 | Documentation & portfolio prep | Pending | README, docs, final polish |

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

### M17 — Student portal
- sqlc query layer (`db/queries/student_portal.sql` + generated code): student profile resolution from authenticated user's person_id, current active enrollment with class/academic year/programme context, full enrollment history across past cohorts, lifetime attendance counts (present/absent/late/excused), results lifetime average/count headline, recent attendance records with joined session/class/subject/term metadata, and student-targeted announcements feed.
- `internal/studentportal` service: read-only, self-scoped aggregation ensuring strict row-level boundary (student account reads only their own personal records; no endpoints accept a student id). Overview aggregation, attendance register listing with limit bounding, results history with optional term/academic-year filters, and full enrollment history.
- HTTP handlers and route wiring in `cmd/server/main.go`, all read-only and gated on the `STUDENT` role:
  - `GET /portal/student/overview`
  - `GET /portal/student/attendance`
  - `GET /portal/student/results`
  - `GET /portal/student/enrollments`
- Unit tests (`internal/studentportal/handlers_test.go`): complete test suite covering overview success/error/unauthenticated, attendance with limit parsing/validation, results with UUID filtering/validation, enrollment history, and missing student profile (404).
- Frontend Next.js integration: `frontend/types/student_portal.ts`, `frontend/services/student_portal.ts`, student portal workspace page `frontend/app/portal/student/page.tsx` with responsive KPI cards, interactive tabs for Overview, Attendance Register, Grades & Results, Enrollment History, and Announcements, plus updated homepage navigation links in `frontend/app/page.tsx`.
- Verification: backend build (`go build ./cmd/server`), full `go test ./...` suite (13 packages passing), and frontend `npm run build` all green.

### M18 — Reports
- `backend/db/queries/reports.sql` and generated code provide the enrollment, attendance, class performance, and report-card aggregates behind the report endpoints.
- `internal/reports/service.go` implements enrollment analytics, class attendance summaries, assessment-based performance reporting, and student report cards with grade calculation and aggregation logic.
- HTTP handlers in `backend/internal/reports/handlers.go` expose the report endpoints and validate input UUIDs/date boundaries.
- Route wiring in `backend/cmd/server/main.go` gates access under the `report.view` permission.
- Unit tests in `backend/internal/reports/handlers_test.go` cover success paths, validation failures, and error handling.
- Verification: backend `go test ./...` passes, including the reports package.

### M19 — Audit logs
- Audit log table and write path already existed from M6; M19 adds the authorized admin read/report endpoints using the existing `internal/audit` package.
- `backend/db/queries/audit_logs.sql` adds `ListAuditLogs`, `CountAuditLogs`, `GetAuditLogByID`, and `ListDistinctAuditActions` while preserving the original `CreateAuditLog` statement.
- `backend/internal/audit/service.go` implements permission-aware read queries, pagination normalization, and filtered list logic for audit review.
- `backend/internal/audit/handlers.go` exposes `GET /api/v1/audit-logs`, `GET /api/v1/audit-logs/:id`, and `GET /api/v1/audit-logs/actions` with structured validation and JSON response shaping.
- The route group in `backend/cmd/server/main.go` is gated with `rbac.RequirePermission(rbacService, "audit.view")` and uses an authenticated admin-only read model.
- Verification: backend `go test ./...` passes after M19 integration.

### M21 — Testing & validation
- Backend regression suite: full `go test ./...` green — 15 test packages (academics, announcements, assessments, assignments, attendance, dashboard, enrollment, parentportal, rbac, reports, staff, studentportal, students, teacherportal) with zero failures.
- Frontend production build: `npm run build` green (8 routes: /, dashboard/admin, portal/parent, portal/student, portal/teacher, reports).
- Live E2E validation against the running docker stack (all HTTP status verified with curl):
  - Auth: login for all four roles (200), bad credentials (401), `/me` (200), logout (204), session invalidated after logout (401 on next call), unauthenticated access to protected endpoints (401).
  - RBAC cross-role denials: teacher→admin dashboard (403), student→audit-logs (403), parent→student portal (403), student→reports (403).
  - Admin: `/dashboard/admin`, `/audit-logs` (paginated), `/audit-logs/actions` — all 200.
  - Reports (`report.view`): enrollment report, class attendance report, class performance report, student report card (requires `term_id` query param) — all 200.
  - Teacher portal (as ada@schoolhub.test): overview, classes, assigned-class roster (200); roster of an unassigned class correctly 403 (`ErrForbiddenClassAccess`).
  - Student portal: overview, attendance (with limit), results, enrollments — all 200.
  - M20 rate limiter: 10 logins/min/IP confirmed live — 429 returned after the limit in a probe run.
- Environment issues found and fixed during validation (none were code bugs):
  - `schoolhub_app` role had never been created in this DB volume — the backend failed to start with a misleading "password authentication failed". Fix: ran `backend/scripts/create_app_role.sql`. NOTE: `psql -v app_password=...` does NOT substitute `:'app_password'` inside the script's `DO $$` block — the password must be inlined (e.g. via sed) until the script is adjusted. Worth a README note.
  - The backend container image was stale (pre-M20 build — rate limiter absent, no 429s). Fix: `docker compose up -d --build backend`. Remember to rebuild the backend image after backend code changes.
  - The three non-teacher seeded accounts had passwords that did not match the documented `ChangeMe123!` (they predate the current seed workflow). Fix: reset their `password_hash` in the dev DB to match the documented dev password.
  - `ada@schoolhub.test` is the TEACHER account (Ada Obi, has a staff row); `teacher@schoolhub.test` (Sarah Mensah) has NO staff row, so its teacher-portal requests correctly 404. There was NO STUDENT-role user at all. Fix: created a dev `student@schoolhub.test` account (ChangeMe123!) linked to student Ama Boateng. All demo credentials: admin@, teacher@ (or ada@), parent@, student@ @schoolhub.test / ChangeMe123!.
  - Port-5432 conflict operational note re-confirmed: start the stack with `POSTGRES_PORT=5433 docker compose up -d` (compose recreating postgres without it will fail to bind).
- Remaining data gaps deferred to M22 (seed/demo environment): `teacher@schoolhub.test` needs a staff profile (or the docs should point at ada@), and realistic demo data volumes for report/portal endpoints.

### M22 — Seed/demo environment
- `backend/cmd/demo-seed/main.go`: a standalone, idempotent demo seeder. Every insert uses fixed UUIDs (extending the existing 11111111/22222222/55555555 demo-skeleton scheme) or the table's natural unique key with `ON CONFLICT DO NOTHING` — safe to re-run, never overwrites API-created data, and recognizes/extends the hand-seeded demo rows from earlier milestones.
- It writes via SQL rather than the service layer on purpose (demo data, not business-flow replay; avoids flooding the audit log with synthetic entries). Runs as the schema-owning role like the migrate tool.
- Dataset: departments (Mathematics/Science/Languages/Humanities) + staff positions; staff rows for Ada Obi (Math+Science, Form 1A/1B) and Sarah Mensah (`teacher@schoolhub.test`, English, Form 1A/1B — M21's missing staff profile closed); 12 Form 1 students across Form 1A/1B with active 2026/2027 enrollments; parent@ linked to both Ama Boateng and Kofi Mensah; class_subjects for all 6 subjects × 2 classes; 5 teacher assignments; attendance for the 20 most recent school days × 6 class-subject combos (119 sessions, 718 records, deterministic PRESENT/ABSENT/LATE/EXCUSED mix); 12 assessments (Quiz 1 + Midterm per combo) with 72 graded results (per-student ability model, letter grades A–F); 5 announcements (published ALL-URGENT / ROLE-TEACHER / CLASS-Form 1A, one DRAFT, one ARCHIVED) authored by admin; self-provisions the `ada@` person/user/staff chain and the `student@schoolhub.test` (Ama Boateng, STUDENT role) account on fresh environments; fixes the pre-existing Ada Obi ON_LEAVE status to ACTIVE.
- Verified: re-run reports +0 rows (idempotent); full E2E suite 33/33 green against the seeded data (all portals, reports, RBAC boundaries, rate limiter); teacher overview shows 2 classes/12 students/80 sessions; parent overview lists 2 children; report card renders attendance breakdown + 77.1% overall average, grade B, per-subject subject list.
- README gained a "Demo environment (M22)" section with the run command, the demo-account table, and updated roadmap/project-state entries.
- Note for later: the seeder requires `cmd/seed`-created admin@/teacher@/parent@ users on a fresh DB (it self-provisions ada@ and student@ itself).

### M23 — Frontend design system & admin shell

Frontend scaffolding milestone — everything later admin screens (M24+) reuse:
- `frontend/lib/api.ts` extended: all HTTP verbs with credentials, 204-safe body handling, and a structured `ApiError` carrying status + backend error code (with `isUnauthenticated`/`isForbidden` helpers).
- `frontend/types/auth.ts` + `frontend/services/auth.ts`: login / me / logout service layer over the M4 session endpoints.
- `frontend/components/auth-provider.tsx`: session context resolving `GET /me` on mount (cookie is the source of truth; no localStorage tokens), exposing user, roles, permissions, login(), logout(). Mounted once in `app/layout.tsx`.
- `frontend/components/protected-route.tsx`: declarative page gating — unauthenticated → redirect to `/login?next=<path>`; wrong role → explicit access-denied panel; loading → spinner. Backend RBAC remains authoritative.
- `frontend/components/app-shell.tsx`: shared layout — role-filtered sidebar (lg+), mobile nav strip, topbar with breadcrumb title + `actions` slot + user chip + sign-out. All five existing screens (admin dashboard, teacher/parent/student portals, reports) refactored onto it, their hand-rolled headers replaced by per-page `actions` chips.
- `frontend/app/login/page.tsx`: real session login with inline error handling and role-based redirect (`?next=` honored).
- `frontend/components/ui/`: reusable kit — `DataTable` (typed declarative columns), `SearchInput` (debounced 300ms, immediate clear), `Pagination` (page/page_size contract), `Modal` (Escape/backdrop dismiss), form primitives (`TextField`, `SelectField`, `TextAreaField`, `FormActions`).
- `frontend/scripts/m23-browser-check.js`: one-off puppeteer-core verification (needs `npm install --no-save puppeteer-core`; uses /usr/bin/chromium). 10 checks: admin login → dashboard under shell with sidebar identity, sign out → login, teacher login → portal renders, teacher denied on admin dashboard (Access denied + ADMIN), bad password inline error, signed-out protected-route redirect. ALL PASS.
- Verification: `npx tsc --noEmit` clean, `npm run build` green (9 routes incl. new `/login`), CORS preflight + credentialed login from the :3000 origin verified against the containerized backend.
- Gotchas learned: `innerText` reflects CSS `text-transform: uppercase` (assert case-insensitively); headless viewport must be ≥1024px to see the lg-only sidebar; networkidle0 can race the client redirect — prefer waitForFunction on the URL.

### M24 — Admin: staff & reference data UI

First consumer of the M23 scaffolding. Three screens:
- `/staff` — directory: DataTable (staff №, name link, department, position, category, status badge, account ✓), debounced search (name/other names/staff number — backend ILIKE), department + employment-status + position-category selects, Pagination (10/page), "Add staff member" modal (all create fields incl. optional department/position/hired date). Gated on `staff.view`; create button on `staff.create`; reference-data link on `staff.update`.
- `/staff/[id]` — profile: personal-details section with Edit modal (PATCH), employment/assignment section with full-replace Edit (PUT, incl. clearing fields), employment-status select in the header (`staff.deactivate`, with server re-sync on error), login-account section — Create-account modal (email/password ≥8/initial role) or Edit-roles modal (checkbox list, wholesale replace). Account UI gated on `hasRole("ADMIN")` because `/roles` is ADMIN-gated (roles fall back to the known system-role list if `/roles` fails).
- `/staff/setup` — departments & positions tables + create modals (`staff.update` for creation, matching the route gates).
- `ProtectedRoute` extended with `requiredPermission` (mirrors the backend's permission-gated routes; role vs permission distinction matters — ADMIN is a role, not a permission).
- AppShell nav gained "Staff Directory" (ADMIN).

**Backend bug found & fixed:** `corsMiddleware` in `cmd/server/main.go` omitted **PUT** from `Access-Control-Allow-Methods`, so every browser preflight for the three M6 PUT routes (assignment, status, account roles) failed with a network error that surfaced as a generic UI error. Fixed to `GET, POST, PUT, PATCH, DELETE, OPTIONS`; backend image rebuilt.

**Backend demo-data extension:** `cmd/demo-seed` now also seeds three extra staff (Grace Amponsah STF-003/Math, Yaw Boateng STF-004/Humanities-Principal, Efua Tetteh STF-005/Science) for list variety (account yes/no, categories).

Verification: `frontend/scripts/m24-browser-check.js` (puppeteer-core + Chromium) — 10/10 PASS: list renders, seeded staff shown, search narrows, staff created via modal, profile renders, account created with TEACHER role, roles updated to TEACHER+STAFF, department created via setup page. Backend `go test ./...` green, frontend tsc + build green.

Gotchas learned: staff endpoints wrap payloads as `{"staff": ...}` (reports/portals use `{"data": ...}`) — service layer must unwrap per domain; the staff list is sorted by name (not recency) and paginated, so a just-created record may land off page 1 (search by staff number to find it); account-roles PUT returns sorted roles (`STAFF, TEACHER`); innerText assertions must tolerate that.

### M25 — Admin: students & guardians UI

Second consumer of the M23 scaffolding, first at the professional quality bar:
- **UI kit upgrades** (reference-repo patterns, see the M25 quality bar note above): `DataTable` zebra striping + row hover + per-column `hideBelow` responsive hiding; `SearchInput` pill style with ring focus; `Pagination` numbered page buttons with compact window + Prev/Next; new `StatCard` (rounded-2xl pastel, context chip, big count) and `Avatar`/`PersonCell` (stable initial-avatars, no photo assets needed).
- **`/students`**: pastel stat cards (active/total/archived), pill search, status filter, 10/page numbered pagination, enroll modal with sectioned form (personal / contact) and inline errors.
- **`/students/[id]`**: personal edit (PATCH), archive/re-activate toggle (`student.archive`) with success feedback, guardians card with contact rows (phone/mail icons from `/icons/`), link-guardian modal (debounced guardian search-and-pick with linked-student counts + relationship select) and unlink with an explicit confirmation modal (explains the portal-access consequence).
- **`/guardians` + `/guardians/[id]`**: directory with linked-children counts, profile with contact details, edit modal, and linked-student cards (relationship, student number, status) linking back to student profiles.
- **Icon assets**: reference repo's 40 PNGs copied to `frontend/public/icons/` — placeholder assets only, swap before M34 deployment (search/phone/mail/view/update/delete/create/logo/logout etc. available).
- Permissions: view on `student.view`, create on `student.create`, edit/link/unlink on `student.update`, archive on `student.archive` — mirroring the M7 route gates exactly.
- Verification: `frontend/scripts/m25-browser-check.js` — 13/13 PASS (directory + stat cards, search, create, profile, link guardian, unlink with confirmation, archive, re-activate, guardians directory, guardian profile with linked children). Backend `go test ./...` green; frontend tsc + build green (13 routes).

Gotchas learned: `text-transform: uppercase` makes innerText uppercase (assert case-insensitively or expect upper); a loading spinner empties the table, so "row disappeared" assertions race — assert the filtered row AND absence together; innerText-based stat checks must match rendered casing.

### M26 — Admin: academic structure UI

- **`/academics`** — tabbed workspace (Academic Years / Terms / Programmes / Subjects / Classes / Class Curriculum), gated on `class.view`, creation on `class.create`, edits/assignments on `class.update`.
- Years/Terms: tables with Current badges, per-row Edit (PATCH, modal) and "Set current" (PUT) actions — the UI copy explains the unique-current-flag semantics (only one current year/term; setting another moves the flag).
- Programmes/Subjects/Classes: tables + create modals (subject type CORE/ELECTIVE select; class form with year/programme selects, level, capacity).
- **Class Curriculum tab**: class selector, assigned-subjects table with assigned_at, subject-assign dropdown (only unassigned subjects listed) + Assign button, and a per-row remove (delete icon) — assign/remove reflect the M8 class_subjects endpoints with success/error feedback.
- AppShell nav gained "Academic Structure" (ADMIN).
- Verification: `frontend/scripts/m26-browser-check.js` — 10/10 PASS (tabs render, current-year flag shown, year created, current flag moved + restored, term created in a picked year, programme created, subject created, curriculum listed, subject assigned, subject unassigned). Backend `go test ./...` green; frontend tsc + build green (14 routes).

Gotchas learned (testing controlled React forms from puppeteer): plain `el.value = …` + bubbling events does NOT update controlled inputs — use the prototype value setters (`HTMLInputElement.prototype`/`HTMLSelectElement.prototype`) then dispatch input+change; entities with unique names/codes collide across test runs — stamp names per run (and remember both code AND name uniqueness); `TextField` needs explicit `id` props for reliable selectors since no `type="text"` attribute is emitted by default.

### M27 — Admin: enrollment & teacher assignment UI

- **`/enrollments` register** — pill filter selects (status/class/year, aria-labelled) + SearchInput + numbered pagination over the date-sorted register. Per-row status-transition `<select>` (gated `class.update`) fires `PATCH /enrollments/:id/status` with optimistic row swap + flash notice / error re-sync.
- Enroll modal: debounced student search-and-pick list (searches name-or-number, 5 results), academic-year select, and a class select filtered by the chosen year (`class.academic_year_id === enrollForm.academic_year_id`) — the front end mirrors the M9 `ErrClassYearMismatch` validation, proved by a fresh student enrolled via search-pick.
- **`/teaching` assignments directory** — teacher/class/subject columns with `PersonCell` links to `/staff/[id]`, search + class filter; create modal loads `listClassSubjects(classId)` on class change so only curriculum subjects can be picked (mirrors M10's `HasClassSubject` chain check); duplicate creates surface the backend's `ALREADY_ASSIGNED` inline; per-row delete icon button with notice.
- Student profile (`/students/[id]`) gained an **Enrollment history** section (class/year/dates/notes, status select on `class.update`, hidden when the student is archived) with its own "+ Enroll in class" modal reusing the year→class dependency.
- AppShell nav gained "Enrollments" and "Teaching Assignments" (ADMIN).
- Backend tweak: the enrollment LIST DTO now includes `student_number` (enrollment detail already had the joins; `enrollments.sql` selects it — the handler simply wasn't mapping it), so register rows can show the number like students/staff screen.
- Verification: `frontend/scripts/m27-browser-check.js` — 13/13 PASS. Backend `go test ./...` green (14 pkgs); frontend tsc + build green (18 routes).

Gotchas learned in M27: the browser checks died on CORS because a stale dev server from an unrelated project occupied port 3000 — the backend whitelists exactly one origin (`FRONTEND_URL`, default `http://localhost:3000`); keep the frontend on 3000 when the demo backend is up. Puppeteer `waitForFunction` assertions must match what the DOM actually renders (student *names*, not numbers — enrollment table cells only show what the row DTO carries; select placeholders live in `<option>`s, label text lives outside the `<select>`). The DB enforces one active enrollment per student+year via a partial unique index covering ENROLLED/**COMPLETED**/TRANSFERRED — status transitions do NOT unlock a same-year re-enroll (the `HasActiveEnrollmentForStudentYear` service check mirrors it); test second enrollments in a different academic year.

### M28 — Attendance UI
- Attendance workspace created at `frontend/app/attendance/page.tsx`, gated under `attendance.view` (shared by `ADMIN` and `TEACHER` roles).
- Types (`frontend/types/attendance.ts`) and API client service (`frontend/services/attendance.ts`) covering all M11 endpoints: `listAttendanceSessions`, `getAttendanceSession`, `createAttendanceSession`, `updateAttendanceRecord`, and `getClassAttendanceSummary`.
- Attendance Directory table: columns for date, class, subject, term, academic year, and row actions ("Inspect & Correct", "Summary"); client search; filters for class, subject, term, date range; numbered pagination.
- KPI Stat Cards: Total Sessions Recorded, Active Classes, Active Terms, Curriculum Subjects Taught.
- "+ Take Register" Modal:
  - Class picker auto-populates class curriculum subjects via `listClassSubjects(classId)`.
  - Filters terms by the class's academic year and defaults to current term.
  - Active enrollee roster loads via `listEnrollments({ class_id, status: "ENROLLED" })`.
  - Individual student status toggles (`PRESENT`, `ABSENT`, `LATE`, `EXCUSED`) with color-coded badges, optional per-student note inputs, and quick-mark helpers ("All Present", "All Absent", "All Late").
  - Form submission to `POST /attendance/sessions` with duplicate conflict handling (`ALREADY_RECORDED` / 409).
- "Inspect & Correct" Modal:
  - Displays session header metadata and summary counts/rates.
  - Renders all student attendance marks with inline `PATCH /attendance/records/:id` updates.
  - Live feedback badge ("Saved!") and reactive recalculation of attendance metrics and present rate.
- "Class Summary" Modal:
  - Displays total records logged, overall present rate percentage, and breakdown counts for present, absent, late, and excused.
- Role-aware navigation: `AppShell` updated to support `roles?: string[]`, displaying the Attendance nav item for both `ADMIN` and `TEACHER`. Homepage updated with Attendance Register card and link.
- Modal component enhanced with optional `maxWidth` prop for wider roster/register views.
- Verification: `frontend/scripts/m28-browser-check.js` — 21/21 PASS across login, directory rendering, filter application, register taking, duplicate rejection, mark correction, class summary, and Teacher RBAC access. Full backend `go test ./...` green (cached/passed across all packages); frontend `npm run build` green (17 static routes).

### M20 — Security hardening
- The limiter is mounted only on `POST /api/v1/auth/login` to blunt brute-force attacks without slowing down the rest of the session-authenticated API.
- `backend/scripts/create_app_role.sql` introduces the least-privilege application role used by the backend instead of the schema-owning migration role.
- `docker-compose.yml` switches the backend service to use the app role via `DATABASE_URL` and keeps the migration/sqlc tooling on the owning `schoolhub` role.
- `.env.example` documents the separate `SCHOOLHUB_APP_PASSWORD` and the operational split between the app role and the migration role.
- Verification: backend `go test ./...` passes and `docker compose config` succeeds cleanly.

---

### Expert pass — professionalization (between M29 and M30)

User direction: make the site look real instead of a demo, integrate Clerk for auth and Resend for email, and "make the website looks professional from now on." What was done:

- **Landing page** (`frontend/app/page.tsx`, rewritten): a full school-website homepage for "SchoolHub Academy" — sticky header with nav + Sign in + Apply now, hero (Unsplash placeholder image, real images come later), stats band, About (image + copy), Academics (Lower/Middle/Senior school cards + why-us list), Admissions CTA band, News teaser cards, and a contact footer (`hello@schoolhub.test`, Accra/Ghana placeholder details). Metadata updated in layout (`title: SchoolHub Academy`).
- **De-demo pass**: milestone-console homepage deleted (was listing M14–M29 launch cards), `login` page lost its "Demo accounts use @schoolhub.test emails" footnote, and the reports page lost its "Milestone M18" context chip. Remaining M-numbers in code files are just comments, not UI.
- **Clerk (optional hybrid bridge)** — the classic M4 session flow is untouched and remains the default when keys are absent:
  - `frontend/lib/clerk.ts` — single knob: `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` set → Clerk mode.
  - `frontend/components/clerk-bridge.tsx` — mounts `ClerkProvider` + a `ClerkSessionBridge` that watches the Clerk session and posts its JWT to the backend bridge the moment one activates (skipping when a backend session already resolves).
  - `frontend/app/login/page.tsx` — renders Clerk's hosted `<SignIn routing="path" path="/login" signUpUrl="/register">` in Clerk mode; legacy email/password form otherwise. `frontend/app/register/page.tsx` hosts `<SignUp/>` ("Registration unavailable" fallback without keys).
  - `frontend/proxy.ts` (Next 16 renamed middleware→proxy) — `clerkMiddleware` + `/__clerk/:path*` matcher, but passes everything through when no key is set so plain deployments are unaffected.
  - Backend: `backend/internal/auth/clerk.go` — `ClerkVerifier` (clerk-sdk-go/v2 `jwt.Verify` + primary-email must be verified) + `Service.LoginWithClerk` reusing the exact session issuance of `Login`; `POST /api/v1/auth/clerk-session` mounted ONLY when `CLERK_SECRET_KEY` is set (404 otherwise, same handler shape as login, not behind loginLimiter). Error mapping: no account → 403 `NO_ACCOUNT`; invalid token → 401 `INVALID_TOKEN` (undifferentiated).
  - AuthProvider gained `refresh()`; `services/auth.ts` gained `createClerkSession()`.
  - `@clerk/nextjs` installed; backend gained `github.com/clerk/clerk-sdk-go/v2`.
  - GOTCHA: the Clerk CLI cannot run on this machine — its prebuilt binary requires AVX and the CPU (Celeron N4020) lacks it (exit 132/SIGILL). Manual integration was used instead; `clerk auth login` + `clerk init` steps from the vendor prompt were skipped as CLI-impossible.
- **Resend email** (optional): `backend/internal/email/email.go` — `email.Service` over Resend's REST API; disabled (no-op) without `RESEND_API_KEY`; a failed send is logged, never fatal. Sends: welcome email on staff account creation (`staff.Service.SetMailer`, fired in a background goroutine after the transaction commits, password never emailed) and school-wide announcement notifications on publish (`announcements.Service.SetMailer`, targets ACTIVE ADMIN/TEACHER/STAFF accounts, only for `TargetType == ALL`, content truncated to 400 chars). Config: `RESEND_API_KEY`, `EMAIL_FROM_ADDRESS` (default `onboarding@resend.dev`).
- **Config surface**: `.env.example` + `frontend/.env.local.example` document all optional keys; `docker-compose.yml` passes `CLERK_SECRET_KEY`/`RESEND_API_KEY`/`EMAIL_FROM_ADDRESS` through from the root `.env` (gitignored) via `env_file: ./backend/.env` + root-env interpolation.
- **LIVE KEYS CONFIGURED (Oct 10)**: Clerk app `epic-pika-7707` (pk_test_… in `frontend/.env.local` alongside `CLERK_SECRET_KEY` — the proxy middleware needs the secret too, else `@clerk/nextjs: Missing secretKey` 500s from proxy.ts; gotcha!). Backend keys in root `.env`. Bridge endpoint confirmed live (401 on junk token); Clerk live browser check `frontend/scripts/clerk-live-check.js` 4/4 (Clerk SignIn replaces the legacy form on /login, register page hosts SignUp, bridge reachable cross-origin). The classic email/password flow is now REPLACED on /login while keys are set — the legacy form only returns when keys are removed.
- **Resend live test**: API key verified — direct API send succeeded (delivered@resend.dev). `onboarding@resend.dev` sender is restricted to the account owner's own email until a domain is verified in Resend (welcome email to a test staff account got 403 — expected; best-effort path logged and continued, account creation unaffected). To unlock real recipient mail: verify a domain in Resend → update `EMAIL_FROM_ADDRESS`.
- Also bumped `backend/Dockerfile` to `golang:1.24-alpine` (go.mod moved to go 1.24.0 when clerk-sdk-go was added). REBUILD the backend image after backend changes: `POSTGRES_PORT=5433 docker compose up -d --build backend`.
- GOTCHAS learned here: (1) `services/auth.ts` passes BARE paths into `lib/api.ts` helpers (they already prefix `/api/v1`) — a rewrite that used full paths produced double-prefix 404s on login; (2) Next 16 prerenders `/register` & `/login` statically — every Clerk hook (`useUser`/`useSession`) must sit behind the same `isClerkEnabled()` gate, including on those pages, else “useSession can only be used within Provider” prerender errors; (3) the stats/uppercase `innerText` gotcha from M25 recurs — assert landing-page copy case-insensitively.
- Verification: `frontend/scripts/expert-pass-check.js` 12/12 (landing renders + no demo text + legacy login → dashboard + register fallback + hero image loads); backend `go test ./...` 14 packages ok + `go vet` clean; frontend `tsc --noEmit` + `npm run build` green (19 routes + Proxy). Bridge endpoint confirmed 404 without keys; live login confirmed 200 after the auth-service path fix.

### Access-control layer — master admin + account blocking (after the expert pass)

User direction: master admin who alone adds/removes other admins; every admin can add/remove parents & teachers (username/email provisioning) and block any parent/teacher **even while signed in**. Discussed and confirmed via multiple-choice: master-admin = a role, blocking reuses `SUSPENDED`, invite step = Clerk invitation preferred (self-register fallback).

- **Migration 000014** (`backend/migrations/000014_master_admin_and_account_control.*`): widens `roles.name` CHECK with `MASTER_ADMIN`, seeds the role + `admin.manage` / `account.block` permissions, grants MASTER_ADMIN every permission (blanket, like ADMIN in 000004), grants plain ADMIN everything except `admin.manage`, and promotes every existing ADMIN user to also hold MASTER_ADMIN (bootstrap invariant: someone can always manage admins).
- **Backend** (`backend/internal/adminaccounts/` — new package, mirroring staff/students layering):
  - `db/queries/accounts.sql` → `ListAdmins`, `ListAccountsWithRole`, `CountAdmins`, `CountByRole` (sqlc regenerated; `backend/db/generated/accounts.sql.go`).
  - `service.go`: `RevokeAdmin` (drops ADMIN+MASTER_ADMIN together; refuses self-demotion, non-admin targets, and removing the last ACTIVE admin — `CountAdmins` guard), `Block`/`Unblock` (flip `users.status` ACTIVE↔SUSPENDED in a tx with an audit row; refuses self-block and any ADMIN/MASTER_ADMIN target — admins are managed by role revocation, not blocking), `ListAdmins`, `ListAccountsByRole`, `IsMasterAdmin`.
  - Audit codes added to `internal/audit/audit.go`: `ADMIN_ACCESS_REVOKED`, `ACCOUNT_STATUS_CHANGED`.
  - Routes in `cmd/server/main.go`: `GET /api/v1/admins` + `PUT /api/v1/users/:id/admin-role` behind `RequirePermission("admin.manage")` **AND** `RequireRole("MASTER_ADMIN")` (double gate so a plain admin granted admin.manage via RBAC customization still can't touch admins); `GET /api/v1/accounts?role=TEACHER|STAFF|PARENT|STUDENT` + `PUT /api/v1/users/:id/status` behind `RequirePermission("account.block")`.
  - **Why blocking works mid-session**: `auth.ValidateSession` (and the Clerk-bridge login path) re-check `users.status` on EVERY request, so a SUSPENDED user's next call gets 401 — proven live: teacher's `/me` went 200 → 401 the moment the admin flipped status, and login again worked after unblock.
- **Frontend**: `types/access.ts` + `services/access.ts` (bare paths), `/access` page (`frontend/app/access/page.tsx`) with two sections — "Administrators" (master-admin only, Remove-admin buttons; hides self/master rows) and "Parent & teacher access" (TEACHER/STAFF/PARENT/STUDENT tab switcher with per-row Block/Unblock, links rows back to `/guardians/[id]` or `/staff/[id]`). AppShell nav gained a 🔐 Access Control item for ADMIN/MASTER_ADMIN. Backend error codes (`LAST_ADMIN`, `SELF_DEMOTION`, `SELF_BLOCK`, `ADMIN_NOT_BLOCKABLE`) surface verbatim per-row.
- **Verification**: `frontend/scripts/access-control-check.js` 15/15 live against the running backend (admin list, master flag, teacher provisioning via `POST /staff` + `POST /staff/:id/account`, block kills live session, unblock restores, all service guards). Backend `go build`/`go vet`/`go test ./...` green (15 pkgs), frontend `tsc --noEmit` + `npm run build` green (20 routes). Test data cleaned (`acct*` users/staff/people removed).
- **Shipped later the same session (Clerk invites + parent-access UI)**:
  - `backend/internal/auth/clerk_invite.go` — best-effort `ClerkInviter` (`POST https://api.clerk.com/v1/invitations` via clerk-sdk-go, 3-attempt retry for flaky first-connect DNS/TLS on this box, tolerant response parsing). Non-fatal by design: provisioning succeeds even when Clerk rejects or no key is set.
  - Wired into BOTH provisioning paths: `staff.CreateAccount` (teachers) and `students.CreateGuardianAccount` (parents) send a Clerk invitation after commit, plus the existing Resend welcome email.
  - `students.CreateGuardianAccount` = `POST /guardians/:id/account` (PARENT role, status ACTIVE, audit `USER_ACCOUNT_CREATED`); rejects duplicate email/existing account; `GetGuardianByID` now joins the account for profile display.
  - Frontend: dedicated **"Create parent access"** button on `/guardians/[id]` (modal with email+password, password min 8), account chip display in the Parent access section, hint text on the staff create-account modal about the Clerk invite email.
  - **Verified**: `frontend/scripts/parent-access-check.js` **14/14** — provisioning, duplicate-email guard, and a REAL pending Clerk invitation (`inv_…`) created via the live API, verified through `GET /v1/invitations`, then revoked in cleanup. Note: Clerk rejects non-real-TLD emails like `example.test` (422) — test with real domains (gmail.com etc.).
- **Browser proof of mid-session block, `frontend/scripts/block-parent-browser-check.js` 11/11**: puppeteer logs admin + parent in through the REAL login form in separate browser contexts, admin clicks **Block** on the parent row in `/access`, the parent's open portal tab is kicked to login on its next request (session 401), and Unblock restores login. GOTCHAS baked into the script: (1) it must run with the dev server in **classic mode** — temporarily move Clerk keys out of `frontend/.env.local` and restart `npm run dev`, because headless Chromium is bot-flagged by **Clerk Protect**, which injects a CSP (`default-src 'self' …` with a nonce) that blocks every client-side fetch to `:8080`; interactive humans pass the silent challenge (proven live), automation does not. (2) Injecting the session cookie via `page.setCookie` did NOT survive navigation — always log in through the form. (3) Restore Clerk mode afterwards (`cp /tmp/env.local.clerk.bak frontend/.env.local` was the manual step).
- GOTCHA: after backend changes, rebuild the image (`POSTGRES_PORT=5433 docker compose up -d --build backend`) — `docker restart` alone re-runs the OLD baked binary (bit this session: routes 404'd until rebuild).

## Current milestone: M34 (user-requested: master-admin & real credentials) — DONE

M34 shipped this session (Oct 14 2026):
- **`must_change_password` mechanism (migration 000015)**: users carry the flag; while set the auth middleware whitelists only `/me`, `/logout`, `/auth/change-password` (+ Clerk bridge endpoints), `POST /auth/login` still succeeds (the client reads the flag off `/me`) and everything else 403s with `MUST_CHANGE_PASSWORD`. `POST /auth/change-password` (classic) verifies the temporary password, stores the new Argon2id hash, clears the flag, and kills every other session for the account; `POST /auth/clerk-password-changed` clears the flag for Clerk-mode rotations. Login responses and `/me` expose the flag.
- **Forced-rotation gate (frontend)**: `frontend/components/password-change-gate.tsx` — ProtectedRoute renders it for flagged sessions BEFORE the role check (flagged users see the gate even on pages their role can't use). Two variants (classic `changePassword` API vs Clerk `user.updatePassword` + `clerkPasswordChanged`, split into separate components so `useUser` is only called inside the Clerk variant — conditional-hook rule). After success: refresh `/me` then `window.location.replace("/")` — the same hard redirect the login page uses, which also fixes the (1) NetworkError complaint (same root cause: stale client-router segments after a session swap).
- **Master admin**: `thunderlight2477@gmail.com` created (people row 'Thunderlight'+'Admin') with password John2477@ + `must_change_password=true`, roles ADMIN+MASTER_ADMIN — **login verified 200, flag verified true**. After the owner's first login rotates it, even the developer can't classic-login as master anymore. The pre-existing Clerk-linked thunderlight2477+admin@gmail.com master remains.
- **`POST /admins` (master-only)**: creates a people row + user with a placeholder password hash (admin accounts can only come alive through the Clerk invitation — classic login with the placeholder always fails), fires the Clerk invite via the same inviter injected into staff's account creation, audit-logs `ADMIN_ACCOUNT_CREATED`. Ordinary admins get 403. The /access "Administrators" section gained the master-admin-only **+ Add admin** composer (email+names, no password field); service/handler wired in main.go alongside the existing access surface.
- **Demo accountability (user chose: delete users, keep rows elsewhere)**: all 10 residual demo/test logins (admin@, opsadmin@, teacher@, ada@, parent@, student@, 4 annparent* residues) DELETED — cascade cleaned sessions/user_roles/reset-tokens, audit_logs actor_user_id SET NULL, announcement author_id reassigned to the master, invitation tokens created by demo users dropped first (RESTRICT FK). Structure rows (people/staff/students/guardians/classes/enrollments) survive untouched.
- **Check-script credentials**: replaced the shared "ChangeMe123!" demo accounts with Q PR personas in the gitignored `frontend/.check-credentials.json` (qa-master ADMIN+MASTER_ADMIN, qa-admin ADMIN, qa-teacher TEACHER on Ada's STF-001 person, qa-parent PARENT on Abena Boateng, qa-student STUDENT on Ama Boateng; all five logins 200). `frontend/scripts/lib/check-accounts.js` is the shared loader (+`login(persona)`); every check script (access-control, parent, parent-access, announcements, audit, reports, m23–m29, block-parent, expert-pass, polish, access-admin) was rewired off hardcoded logins. audit-check.js smoke run: **16/16 API checks green** (browser leg fails in Clerk mode because /login has no #email form — run browser scripts in classic mode as before).
- **Seeded-demo credentials retired from code too**: `cmd/demo-seed` no longer creates any login users (Sarah's person resolved by name, Ada by STF-001, guardian link by guardian person name, announcement author = the real master) and `cmd/seed`'s doc examples lost the ChangeMe123! pattern. Re-ran `demo-seed` live: idempotent, config intact, **0 demo accounts created**.
- **Verified**: backend `go build ./...` + `go vet ./cmd/demo-seed/` clean; frontend `tsc --noEmit` + `npm run build` green; access-admin-browser-check **8/8** (master composer, invite-only modal, plain-admin exclusion, forced gate on a flagged QA teacher, restored after); dev server left running in Clerk mode (login 200, clerk refs). QA creds NEVER commit — .gitignore covers .check-credentials.json.
- **Gotchas learned in M34**: `users` cascades everything safe EXCEPT `invitation_tokens.invited_by` (RESTRICT — null/delete first) and `audit_logs.actor_user_id` (SET NULL); student/teacher portal identity resolves user→people→students/staff by person_id, so QA personas must attach to the person row behind an existing staff/student/guardian row for realistic portal data; Clerk middleware (`proxy.ts`) requires a Clerk session for every non-public route — backend-only cookies get bounced, so seed browser sessions via API cookie injection only in classic mode; `min(uuid)` doesn't exist (ORDER BY created_at LIMIT 1 instead); puppeteer-core + /usr/bin/chromium already set up on this box.
- **M34 governance completion (user chose "implement all three", teacher surface = reuse /guardians)** — verified 14/14 API + 9/9 browser:
  - Migration `000016_user_invited_by`: `users.invited_by UUID NULL REFERENCES users(id)` + index + the `guardian.create` permission granted to TEACHER/ADMIN/MASTER_ADMIN. GOTCHA: `schema_migrations` can be stuck `dirty` from an earlier interrupted run (000015 was applied but flagged dirty) — clear `dirty=false` only after verifying the version's SQL is fully present, then `docker compose run --rm migrate up`.
  - `GET /my-invitees` (RequireAuth only — access control IS the `invited_by = caller` filter) + `PUT /my-invitees/:id/status` (inviter-scoped suspend/restore, 403 `NOT_YOUR_INVITEE` otherwise). `setAccountStatus` now permits admin targets when the caller is MASTER_ADMIN or the target's inviter, with the `countActiveMasterAdminsExcept` last-master guard (reachable only via a demoted-inviter edge case, but guarded anyway). Login of a suspended account answers **403 ACCOUNT_NOT_ACTIVE** (deliberate, predates M34 — checks must expect 403, not 401).
  - Guardians routes split: `POST /guardians` moved from `student.create` to `guardian.create`; `POST /guardians/:id/account` moved from `staff.update` to `guardian.create` (admins keep both codes via the grant). Frontend: /access gate dropped to plain auth with a "People you invited" section for everyone; nav items `/access` + `/guardians` now include TEACHER; `canCreate`/`canProvision` flipped to `guardian.create`.
  - **Latent bug found & fixed**: `GET /admins` 500s (`cannot scan NULL into *string`) the moment a plain ADMIN exists — the `is_master` scalar subquery returns NULL and sqlc typed it non-null. M34's CreateAdmin is what made plain admins possible (before, migration 000014 promoted everyone to MASTER). Fix: `COALESCE((SELECT …), '') AS is_master` in `db/queries/accounts.sql` + sqlc regen. Never surfaced earlier because checks ran while only masters existed.
  - Checks: `frontend/scripts/m34-governance-check.js` (14/14) and `frontend/scripts/m34-governance-ui-check.js` (9/9, classic mode); `frontend/scripts/crossrole-repro-check.js` (4/4 — sign-out NetworkError cannot be reproduced, AppShell signs out via `window.location.replace`); `frontend/scripts/m34-lastmaster-check.js` (4/4 — stages the demoted-inviter edge live: invite → promote to MASTER → suspend the real masters → demote the inviter → inviter's suspend attempt gets 409 LAST_MASTER, then restores everything). Backend `go test ./...` green, frontend tsc+build green.
  - **Guard bug found by that staging**: the last-master count check originally sat inside `if actorIsMaster`, and the role loop `break`-ed on the first match (ADMIN sorts before MASTER_ADMIN) so `targetIsMaster` never got set — dead code. Restructured: single pass computes `targetIsAdmin`/`targetIsMaster` flags first, then the admin-target gate, then the master-suspension count guard runs for **every** actor. GOTCHA: `docker compose up -d --build` from `backend/` cwd silently skips the rebuild — run it from the repo root and compare `docker inspect … .State.StartedAt` before/after (a stale binary was misdiagnosed as a logic failure for two rounds).
  - **Classic-mode skip workflow (fix for "browser legs fail in Clerk mode")**: `frontend/scripts/lib/check-accounts.js` exports `isClassicMode()` / `requireClassicMode(script)` (greps served /login for `clerk.accounts.dev`); EVERY script with a form-driven browser leg is now gated on it — in Clerk mode they print an explicit `SKIPPED — <name> browser leg requires classic mode …` line and exit 0 without running the leg (API checks in mixed scripts still run and still count, e.g. audit-check 17/17+SKIP); in classic mode nothing changes (audit-check 18/18, crossrole 4/4, governance-ui 9/9 re-verified).

## M34 pre-session planning notes (superseded by the DONE record above)
- Cross-role bug: admin → sign out → login as teacher/parent throws `Runtime TypeError: NetworkError when attempting to fetch resource` (Next 16.3.7/Turbopack); a refresh logs in fine. Investigate: likely stale client-router/RSC cache or prefetch of the previous role's page right after session swap (auth-provider/Clerk bridge), possibly a global-error overlay from a failed prefetch RSC fetch.
- Master admin account: create user thunderlight2477@gmail.com (MASTER_ADMIN) with flag forcing password change on first login (store initial password John2477@ hashed; prompt+verify old, set new); backend needs a must_change_password flag/mechanism, frontend needs the change dialog gate.
- "Add admin" UI: check backend /access surface (`admin.manage`) — currently only block/unblock/revoke exist; may need a create-admin endpoint (whitelisted invitation path). Ordinary admin UI should hide master-admin-only controls.
- Demo details: remove seeded test emails/passwords from the DB and any demo hints in the UI; the identity-check scripts (audit-check, parent-check, reports-check…) depend on those test accounts and will need new credentials or a dedicated seed-mode document.

## M33: Frontend polish & responsive pass (done)

- **Accessible shared Tabs** (`frontend/components/ui/tabs.tsx`): role=tablist/tab, aria-selected, roving tabindex, ArrowLeft/Right/Home/End with focus-follows-selection; two visual variants (underline = content tabs, pills = segmented control). Refactored: `/reports` (4 tabs), `/academics` (6 pills), `/portal/student` (5 tabs), parent portal child-detail (Attendance/Results). Teacher portal had no tab row.
- **Modal a11y** (`components/ui/modal.tsx`): focus moves into the dialog on open (first focusable), previous focus restored on close, `aria-labelledby` via useId.
- **Form a11y** (`components/ui/form.tsx`): TextField/SelectField/TextAreaField auto-generate a stable `id` from the label via useId when callers omit one → labels always link; `aria-invalid` + red border styling when `error` is set. GOTCHA: object-literal props must QUOTE hyphenated keys ("aria-selected") — TS1005 parse errors otherwise.
- **AppShell**: `aria-label="Primary"` navs + `aria-current="page"` on sidebar/mobile-strip links; topbar row now wraps (`min-h-16 flex-wrap`, lg restores h-16/nowrap) — fixes /staff's 405px actions row overflowing 390px (+47px). DataTable already scrolls internally (overflow-x-auto), the only true document-level offender.
- **Verified** `frontend/scripts/polish-check.js` **41/41** (classic mode): 3 form logins (admin/parent/student — GOTCHA: all browser.newPage() pages share ONE cookie context, so the LAST formLogin's session wins; inject the intended role's cookies explicitly per probe); overflow sweep 13 admin routes + both portals × desktop+390 with `scrollWidth - innerWidth <= 2`; empty-state confirmed built into DataTable; tabs a11y + ArrowRight selection on /reports (4 tabs) and student portal (5 tabs); audit detail modal focuses "Close" inside dialog + Escape closes; announcements composer's 5 labels all resolve htmlFor→id.
- Loading/empty/error states were already present on every page (recon grep counts); DataTable ships a built-in zero-row empty state, so no PageState primitive was needed.
- Dev-server env note: /tmp/env.local.clerk.active stash no longer exists — stash to a session-specific name (e.g. /tmp/env.local.clerk.m33) and restore it when done.

## M32: Audit log viewer UI (done)

- **Frontend over the M19 backend** (all three `GET /audit-logs*` endpoints, gated in main.go by `RequireAuth` + `RequirePermission("audit.view")` — handled by ADMIN and MASTER_ADMIN roles): `frontend/types/audit.ts` + `frontend/services/audit.ts` (bare paths: `listAuditLogs`, `getAuditLog`, `listAuditActions`) and `frontend/app/audit/page.tsx`. AppShell nav gained 📋 Audit Log (ADMIN + MASTER_ADMIN roles).
- Page: banner + StatCards (total entries from pagination.total, distinct actions from /actions, current page / total pages), filter bar (action dropdown fed by `GET /audit-logs/actions` — the action set grows every milestone so it must NOT be hardcoded client-side; static dropdown for the 16 distinct `resource_type` strings; date_from/date_to date inputs converted to **RFC3339** with `T00:00:00Z`/`T23:59:59Z` suffixes before hitting the API), DataTable (When/By/Action/Resource + View button, action chips colour-coded by category via name match — DELETED rose, CREATED emerald, ARCHIVED/SUSPENDED amber, PUBLISHED/SET_CURRENT sky), Pagination (page_size 20), and a detail modal rendering the decoded `metadata` JSONB via `JSON.stringify(…, null, 2)` (backend guarantees `{}` on decode failure, fail-soft).
- `ProtectedRoute requiredPermission="audit.view"` — **use the permission gate, not `requiredRole="ADMIN"`**: `hasRole` in auth-provider is a plain `roles.includes()`, no role hierarchy, so a MASTER_ADMIN-only user would fail an ADMIN role check; permissions come from `/me` and hold for both admin roles.
- **Verified** (`frontend/scripts/audit-check.js`, **18/18**, mirrors reports-check.js): admin+teacher logins; list envelope shape (`{audit_logs, pagination}`); DTO fields (RFC3339 `created_at`, metadata object); /actions returns 32 sorted distinct codes; detail get + 404; action/resource_type/actor_user_id filters all filter; date window (2026 wide → 233, 2025 narrow → 0); pagination mechanics (page_size=2, pages differ, 117 total pages); validation 400s (bad actor UUID, **bare-date `date_from` rejected — RFC3339 required**, non-integer page); teacher **403** (audit.view is admin-only — unlike M30's create-endpoint gotcha, denial checks are VALID for view-gated routes); unauth 401; browser render of `/audit` via classic-mode form login (banner + filters + seeded action chips).
- **GOTCHAS learned:**
  1. **Stored `resource_type` values are LOWERCASE** (`academic_year`, `announcement`, `attendance_record`, … — 16 distinct strings as of M32). The page's static dropdown was first written UPPERCASE and would have filtered to zero rows; fixed to the live values. If backend Record ever writes new resource types, update both the static list and this note — there is no /audit-logs/resource-types endpoint.
  2. The `.next/dev/types/validator.ts` file can be left corrupted/truncated (TS1128 errors) by the killed dev servers from the M31 session — `tsc` fails on repo code that is actually fine; `rm -rf .next` and rerun before suspecting real type errors.
  3. TextField is a plain-input wrapper: its `onChange` receives the **event**, not the value — `{onChange={(e) => setState(e.target.value)}}`, unlike SearchInput which hands the raw string to onSearch.
  4. Stale `.env.local` state: after a classic-mode swap, `.env.local` must be restored from `/tmp/env.local.clerk.active` (this session) before browser steps in Clerk mode. Both M31 and M32 ended with dev server restarted in Clerk mode (login 200 + clerk.accounts.dev refs present) via `setsid nohup npx next dev --port 3000` — that setsid invocation is the only reliable way on this box.

## M31: Reports UI (done)

- **Frontend over the M18 backend** (all four `GET /reports/...` endpoints, gated server-side by `report.view`): `frontend/types/reports.ts`, `frontend/services/reports.ts`, and the four-tab `/reports` Reports Center page (`frontend/app/reports/page.tsx`) — nav item 📈 Reports Center already existed in AppShell for all roles.
  - **Enrollment Summary tab**: loads `GET /reports/enrollment` on mount; KPI cards (total enrolled, capacity utilization with `(enrolled/capacity)` sub-label, gender split, academic year) + per-class table (utilization mini-bars, green/amber/red over 85/100) and `academic_year_id` filter hook.
  - **Class Attendance tab**: class select (first-3 quick-pick chips fed by the enrollment pull, `term_id`/`start_date`/`end_date` optional filters) → `GET /reports/attendance/classes/:id`; per-student counts + rate chip, class-average KPI.
  - **Academic Performance tab**: class + optional `term_id`/`subject_id` → `GET /reports/performance/classes/:id`; assessment rows with avg/high/low, percentage of max.
  - **Student Report Card tab**: student UUID + required `term_id` → `GET /reports/students/:id/report-card`; printable card (header, per-subject grade rows, attendance strip) with `window.print()`.
- **Verified** (`frontend/scripts/reports-check.js`, **15/15**, mirrors `announcements-check.js`): API checks against seeded data — enrollment structure + Form 1A utilization (seeded IDs: Form 1A `55555555-…5555`, Term 1 `22222222-…2222`, student Ama Boateng `bbbb0001-…0001`); attendance (61 sessions, 6 students, avg 85%); performance (13 assessments); report card (8 subjects, overall B); error paths (unknown class 404 ×2, unknown student 404, missing `term_id` 400, bad-UUID 400s ×2, invalid date range 400 — start after end), unauthenticated 401; browser check renders `/reports` through the classic-mode login form (tabs + Form 1A rows present).
- **tsc --noEmit + npm run build green** (`/reports` among the route list). No backend changes: M18 DTOs were already exactly matched by the existing types file — this session only verified them end to end and wrote the check script.
- **GOTCHAS learned (box-level, recurring)**:
  1. **Dev server must start via `setsid nohup … &`** — a plain `nohup npx next dev … & disown` child dies the moment the tool-call shell session is torn down (every cross-call death this session traced to this, NOT OOM: `/proc/vmstat` `oom_kill` stayed 0). remember.md's runbook already said this; it is load-bearing.
  2. **Never `pkill -f "next dev"` / `pkill -f "npm run dev"` / `pkill -f next-server` from a tool call or script on this box** — `-f` matches the full command line, which includes the invoking wrapper shell's own arguments, so the kill often takes out the session that launched the replacement server. Kill by PID instead: `PID=$(ss -tlnp | grep 3000 | grep -oP 'pid=\K[0-9]+' | head -1); kill $PID`. Same applies inside check scripts — restart logic there must not pkill by name either.
  3. `free -m` on this box constantly shows ~100 MB free + heavy swap under VS Code load — that alone is NOT proof the dev server was OOM-killed; check `oom_kill` counter first before blaming memory, and identify the real killer by PID-watching (`while kill -0 $PID; do sleep 1; done`).
  4. Gotcha (2) from M30 recurs: classic-mode detection via `grep -c "clerk.accounts.dev"` on /login HTML; `#email` only exists in classic mode. Restore Clerk mode afterwards (this session: `.env.local` cycled via `/tmp/env.local.stash` → restored from `/tmp/env.local.clerk.active`; dev server left running in Clerk mode, login 200 + clerk refs present).

## M30: Announcements composer UI (done)

- **Frontend only** — the announcements backend (M13) was already complete, so M30 = the UI layer: `frontend/types/announcements.ts` + `frontend/services/announcements.ts` (bare paths, apiGet/apiPost/apiPatch/apiDelete) and `frontend/app/announcements/page.tsx`.
- Composer modal drafts as **DRAFT** always (`status: "DRAFT"` in create payload) — publish is an explicit table action; edit reuses the modal via PATCH. Targeting: ALL / ROLE (TEACHER/STAFF/PARENT/STUDENT) / CLASS (classes lazily fetched from `/classes` on first CLASS selection). Priority chips, optional expiry date. Publish/Archive/Delete per row gated on `hasPermission("announcement.update")`; New Announcement button on `announcement.create`; delete has a confirm modal.
- Directory: StatCards, SearchInput (debounced, title/content via `search=`), status + audience selects, DataTable + Pagination (page_size 10). AppShell nav gained 📢 Announcements (ADMIN role). Backend filters/rejection live through M13 routes untouched.
- **Verification** (`frontend/scripts/announcements-check.js`, **15/15**): admin login → create draft → filter-list it → PATCH edit → publish → feed (`?feed=true`) shows it to a provisioned PARENT (ALL target) → ROLE=PARENT targeted announcement also reaches the parent feed → archive → delete → 404 after delete → unauthenticated create rejected 401 → browser check that `/announcements` renders with the composer and the new draft (drives the classic login form through puppeteer).
- **GOTCHAS**: (1) TEACHER holds `announcement.create` by design (M13 migration 000004 header — "class-scoped announcements"); the composer being admin-only is an M30 UI product decision via nav/permission gating, so a check "teacher 403 on create" is WRONG — the API grants 201. Use the unauthenticated-401 check instead. (2) Classic-mode detection for browser scripts: `grep -c "clerk.accounts.dev"` on the fetched /login HTML — Clerk mode shows refs, classic shows 0; `#email` selector only exists in classic mode. (3) Node resolves `puppeteer-core` only from the script's own directory tree — scripts under `frontend/scripts/` get it from `frontend/node_modules`, but ad-hoc probes in /tmp need `NODE_PATH=frontend/node_modules` or an absolute require. (4) After an env swap + dev-server restart, the first `curl` can hit before Turbopack cold-compiles — poll (`ss -tln` on 3000, retry the fetch) instead of one-shot sleeps.

## M13 backend context (fed by the M30 UI)

The repo contains the completed milestone stack through M30 + the expert pass:
- Assessment creation header (`POST /assessments`) with class, subject, term, type (HOMEWORK, QUIZ, TEST, EXAM), max score, weightage.
- Assessment score sheet grid / bulk entry (`POST /assessments/:id/results/batch`).
- Single result correction (`PATCH /results/:id`).
- Assessment list directory with filters (class, subject, term) and pagination.
- Gated under `assessment.view`, `assessment.create`, `assessment.update`, `result.view`, `result.create`, `result.update`.

- M23: frontend design system & admin shell complete
- M24: admin staff & reference data UI complete
- M25: admin students & guardians UI complete
- M26: admin academic structure UI complete
- M27: admin enrollment & teacher assignment UI complete
- M28: attendance UI complete
- M29: assessments & results UI complete
- M30: announcements composer UI complete
- Expert pass: professionalization (landing page, de-demo, Clerk bridge, Resend) complete
- Post-M30: access-control layer (MASTER_ADMIN, blocking, `/access`), Clerk invitations, "Create parent access" button — all complete

Operational note: another project's container (`lightex-db`) occupies host port 5432 in this environment. Run the schoolhub stack with `POSTGRES_PORT=5433 docker compose up -d` and point the local server at `postgres://schoolhub:schoolhub@localhost:5433/schoolhub?sslmode=disable`. The containerized backend is unaffected (internal network).

Operational note (dev servers): only ONE dev server should run for this project, on port 3000 — the backend's CORS whitelist is a single exact origin (`FRONTEND_URL`, default `http://localhost:3000`) and requests from any other port (e.g. 3001 when Next auto-increments because 3000 is busy) are rejected. If 3000 is taken by an unrelated project's server (it already happened once — `/mnt/grace/royal-solwezi-lodge-1` served a lodge site on 3000), stop that process first; do not let the schoolhub dev server fall back to another port. Start it from `frontend/` with `setsid nohup npm run dev > /tmp/next-dev.log 2>&1 < /dev/null &`.

Operational note (DB drift): test runs can leave residue — orphan students/enrollments (names like `M27-*`, `QA *` years with no classes), stub staff (`M24-*` numbers), extra classes, and their audit rows. Before a demo or a run whose assumptions depend on clean state, verify `students=12, enrollments=12 (all ENROLLED, year 2026/2027), staff=5, teacher_assignments=5 (Ada×4 + Sarah Form 1A ENG), classes=2 (Form 1A/1B in 2026/2027), academic_years: 2026/2027 current` and run `cd backend && DATABASE_URL='postgres://schoolhub:schoolhub@localhost:5433/schoolhub?sslmode=disable' go run ./cmd/demo-seed` to restore.

---

## Working rules for future milestones

- Implement one milestone at a time.
- Keep discipline: migration, sqlc, service, handlers, routes, tests.
- Do not jump ahead into later domains.
- The project remains a modular monolith with clear domain boundaries.
- Any change to architecture must be explained before it is adopted.
- Frontend work is local validation/demo scope unless a milestone explicitly includes UI work.

---

## Quick operational notes

- Main backend: backend/
- Main frontend: frontend/
- DB layer: backend/db/
- Migrations: backend/migrations/
- App entrypoint: backend/cmd/server/main.go
- Most recent completed milestones: M30 announcements UI, M31 reports UI, M32 audit UI, M33 polish pass, M34 master-admin & real credentials
- Current next milestone to implement: M35 deployment

## Recommended continuation pattern

When starting the next milestone, use this sequence:

1. Confirm the domain requirements and migration impact
2. Add or update SQL queries
3. Regenerate sqlc output
4. Implement service logic
5. Add handlers and route wiring
6. Test the endpoints and business flows
7. Update this file and the repo docs with the new milestone status

