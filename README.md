# SchoolHub

A full-stack school administration & management portal built as a portfolio
project that demonstrates practical engineering: Next.js + TypeScript on the
frontend, Go on the backend, and PostgreSQL for persistence.

> Status: Post-expert pass with access-control layer (master admin + account
> blocking) done. The backend includes the full identity/RBAC foundation,
> auth, staff, students and guardians, academics, enrollment, teacher
> assignments, attendance, assessments/results, announcements, portals,
> reporting, plus master-admin access control with immediate mid-session
> account blocking.

> Scope note: the frontend is a local demo and validation layer for the API. It
> is not considered a milestone acceptance requirement unless a milestone
> explicitly includes UI work.

**This is a personal portfolio/recruitment project.** It is not commissioned,
endorsed, or used by any specific school, and it contains fictional data only.

## Current project state

The repository is now at the end of M18 and the backend is materially ahead of
an earlier milestone-only roadmap. The codebase includes:

- project scaffolding and local container setup
- migrations from 000001 through 000014
- sqlc-generated database access code
- authentication, sessions, and role/permission enforcement
- people/staff, students, guardians, and academic structure domains
- enrollment, teacher assignments, attendance, assessed results, and announcements
- admin, teacher, parent, and student portal domains
- institutional reports with `report.view` permission enforcement
- a least-privilege app DB role plus login rate limiting (M20)
- an idempotent demo-data seeder (`backend/cmd/demo-seed`) for structure data —
  no seeded login accounts (M34 removed shared demo credentials)
- a frontend auth + design-system layer: login page, session auth context,
  protected routes, shared app shell, and a reusable UI kit (M23)
- a staff management UI: directory with search/filters/pagination, staff
  profiles with assignment/status/account management, and department &
  position administration (M24)
- a students & guardians management UI: student directory with stat cards,
  profiles with archive/re-activate and guardian link/unlink, and guardian
  directory/profile with linked children (M25)
- an academic structure UI: tabbed workspace for years/terms with
  current-flag control, programmes, subjects, classes, and per-class
  curriculum assignment (M26)
- an enrollment & teaching assignments UI: enrollment register with filters
  and status transitions, search-pick enrollment modal, teaching-assignment
  directory constrained to each class's curriculum, and per-student
  enrollment history (M27)

## Professionalization pass (landing page, Clerk, Resend)

Beyond the milestone stack, the product front door has been made production-flavored
rather than demo-flavored:

- **Real school landing page** — the homepage is now the public website of
  "SchoolHub Academy": hero with photography, stats band, about, academics
  (Lower/Middle/Senior school), admissions call-to-action, news teaser, and a
  contact footer. Placeholder imagery comes from Unsplash; swap in real
  photography before launch.
- **Demo text removed** — milestone-console cards, "demo account" hints in the
  login UI, and milestone chips no longer appear in any user-facing screen.
  (Shared demo credentials were removed from the DB — see the M22 section.)
- **Optional Clerk authentication (hybrid bridge)** — set
  `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` (frontend) and `CLERK_SECRET_KEY`
  (backend) and `/login` renders Clerk's hosted sign-in instead of the
  email/password form; after a Clerk sign-in the app posts the Clerk session
  JWT to `POST /api/v1/auth/clerk-session`, which verifies it server-side and
  issues the app's standard session cookie. RBAC, roles, audit, and every
  downstream screen are unchanged. Without keys, classic session auth is used
  and all Clerk code paths are inert. Accounts are never auto-created from a
  Clerk identity — the verified email must match an account provisioned by an
  admin.
- **Optional Resend email** — set `RESEND_API_KEY` and (optionally)
  `EMAIL_FROM_ADDRESS` on the backend and outbound email turns on: a welcome
  email when an admin provisions a staff login account, and a notification
  email to staff/admin account holders when a school-wide announcement is
  published. Every send is best-effort and logged, never fatal; the app is
  fully functional without the provider.

All integration keys are documented in `.env.example` and
`frontend/.env.local.example`.

## Access-control layer (master admin & account blocking)

On top of the classic RBAC model there is a two-tier administration model and
an account-lifecycle control:

- **Master admin (MASTER_ADMIN role)** — sits above ADMIN. Only master admins
  can list administrators (`GET /api/v1/admins`), **invite other admins**
  (`POST /api/v1/admins` — Clerk invitation, no password ever typed), and
  revoke another user's admin access (`PUT /api/v1/users/:id/admin-role`),
  gated by both the `admin.manage` permission AND `MASTER_ADMIN` role
  membership. Regular admins keep every other capability but cannot
  manufacture or demote other admins.
  Service-level guards refuse removing the **last active admin** (locking the
  school out) and **self-demotion**. Every grant/revoke is audit-logged
  (`ADMIN_ACCESS_REVOKED`).
- **Blocking accounts** — any admin (via the `account.block` permission) can
  suspend or restore a teacher/staff/parent/student login from the Access
  Control screen. Blocking flips `users.status` to `SUSPENDED`; because
  `ValidateSession` re-checks status on **every** request, the lockout is
  immediate even in an active browser session, and the same check governs the
  Clerk bridge path. **Admin accounts can be blocked too (M34)** — by a
  MASTER_ADMIN or by the account's own inviter — while revoking the ADMIN
  role remains the way to strip privileges; the service refuses to suspend
  the last ACTIVE master admin (lockout guard).
- **Inviter tracking (migration 000016)** — every provisioned login stores
  `users.invited_by`, powering two governance guarantees:
  `GET /api/v1/my-invitees` shows each caller **only the accounts they
  personally minted** (admins, staff, parents), and
  `PUT /api/v1/my-invitees/:id/status` lets **any** inviter — teachers
  without `account.block` included — suspend/restore exactly those rows
  (anything else → 403 `NOT_YOUR_INVITEE`).
- **Teachers invite parents (M34)** — the `guardian.create` permission
  (held by TEACHER + both admin roles) gates guardian creation and parent
  account provisioning, so teachers use the same `/guardians` screens;
  student minting and staff accounts stay admin-only.
- **UI** — the `/access` screen (sidebar: 🔐 Access Control, visible to
  admins) shows an "Administrators" table with a master-admin-only
  **+ Add admin** composer (email + name, invite-only — no password field),
  and a TEACHER/STAFF/PARENT/STUDENT tab view with Block/Unblock per row, for
  any admin. Rows link back to the guardian/staff profile the account belongs to.
- **Forced first-login password rotation (M34)** — sessions carry a
  `must_change_password` flag; while it is set the auth middleware lets only
  me/logout/change-password through, and every protected page renders the
  "Set your own password" gate (classic mode: POST /auth/change-password;
  Clerk mode: Clerk-side rotation then POST /auth/clerk-password-changed).
  Newly invited admins (and the initial master) cannot use the workspace
  until they rotate away from the temporary password, after which even the
  issuer can't reuse it.
- **Invitations** — creating an account for a teacher (staff profile →
  "Create account") or a parent (guardian profile → **"Create parent access"
**) provisions the SchoolHub user, sends the Resend welcome email, and
  best-effort issues a **Clerk invitation** (`POST /v1/invitations`) so the
  invitee gets Clerk's official "set your password" email. Provisioning
  never fails because of Clerk (invitation errors are logged, not raised).
- **Browser-verified** — `frontend/scripts/block-parent-browser-check.js`
  (run with the dev server in classic mode, i.e. without Clerk keys) drives
  real Chromium sessions: an admin clicking Block in `/access` kicks a
  signed-in parent's open portal tab on its next request, and Unblock lets
  them sign back in. 11/11 checks.
- Schema: migration `000014_master_admin_and_account_control` seeds the role,
  the two permission codes, and promotes pre-existing admins to master admin.

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

## Deploying the demo (M35)

This is the go-live recipe for a single-school **demo deployment** — show a
school's decision-makers a working system, then build the real thing if they
agree. Topology: **Cloudflare** hosts the Next.js frontend (Workers via the
OpenNext adapter — current supported path for Next 16; classic Pages next-on-pages
predates it), **Render** (free tier) runs the Go API from `backend/Dockerfile`
via the `render.yaml` Blueprint, and **Neon** (free tier) provides Postgres.
Classic email/password sessions are used — no Clerk account or keys needed.

Order of operations (each step gates the next):

**1. Neon — create the database.** Sign up at neon.tech, create a project
(`schoolhub`), and copy the **pooled** connection string (endpoint contains
`-pooler`, connection mode "Pooled"). Append `&sslmode=require` if not
present. This is the `NEON_DB_URL` used in every later step.

**2. Apply the schema** (golang-migrate image, no local install needed):

```bash
docker run --rm -v "$PWD/backend/migrations:/migrations" \
  migrate/migrate:v4.17.1 \
  -path=/migrations -database="$NEON_DB_URL" up
```

**3. Bootstrap the school's master admin** (from `backend/` — `cmd/seed`
creates ADMIN and now also grants MASTER_ADMIN on a fresh database, the
bootstrap invariant). Pick the school's real email and a strong password:

```bash
database.NewPool is used by cmd/seed; set DATABASE_URL directly:
DATABASE_URL="$NEON_DB_URL" go run ./cmd/seed \
  -email head@yourschool.example -password '…' -first-name Ada -last-name Obi
```

The printed password gate applies: the first login forces a password change
(M34) and afterwards the seeded password can never be reused.

**4. Load the demo dataset** (`cmd/demo-seed` is idempotent; the announcement
author is now resolved as the oldest MASTER_ADMIN instead of a hardcoded
email):

```bash
DATABASE_URL="$NEON_DB_URL" go run ./cmd/demo-seed
```

**5. Render — deploy the API.** In the Render dashboard, "New + > Blueprint",
point it at this repository. `render.yaml` creates `schoolhub-api` (Docker,
free plan, health check `/api/v1/health`). In the service's Environment tab
fill in the two `sync: false` secrets:

- `DATABASE_URL` = `$NEON_DB_URL` (the pooled string from step 1)
- `FRONTEND_URL` = the exact deployed frontend origin (set this again after
  step 6 gives you the URL, then save — CORS echoes it verbatim)

`APP_ENV=production` + `COOKIE_SAMESITE=none` are already set in the
Blueprint (the Pages/Workers frontend and this API sit on different
registrable domains, and a Lax cookie set in a cross-site response is
dropped by the browser; `config.Load` enforces that `none` runs only in
production).

**6. Deploy the frontend to Cloudflare** (from `frontend/`):

```bash
# FIRST deploy: set your Render URL as the baked-in API base
NEXT_PUBLIC_API_URL=https://schoolhub-api.onrender.com/api/v1 \
  npx wrangler login
NEXT_PUBLIC_API_URL=https://schoolhub-api.onrender.com/api/v1 npm run build:cf
NEXT_PUBLIC_API_URL=https://schoolhub-api.onrender.com/api/v1 npm run deploy:cf
```

This deploys to `schoolhub-frontend.<subdomain>.workers.dev`. For Git-based
auto-deploys instead, create a Worker in the Cloudflare dashboard with
framework preset "Next.js" and add `NEXT_PUBLIC_API_URL` as a build variable
(it must exist at BUILD time — it is compiled into the client bundle).

**7. Close the loop.** Back on the Render `schoolhub-api` Environment tab,
set `FRONTEND_URL` to the final frontend URL from step 6 and save (this
redeploys the API). Then verify end-to-end:

- `curl https://schoolhub-api.onrender.com/api/v1/health` → `{"status":"ok"}`
- Open the frontend URL, sign in as the step-3 master admin, change the
  password when gated, and walk the demo school (staff → students →
  academics → attendance → assessments → results → reports → announcements
  → audit).

**Demo-environment notes**

- Neon free tier & Render free tier: the API sleeps after ~15 minutes
  without traffic; the first request (including the health check) takes a
  few seconds while it cold-starts. Refresh once.
- Resend is left unset (`RESEND_API_KEY` empty in the Blueprint) — email is
  fully optional; announcement/provision flows work without sending.
- Clerk stays disabled for the demo (`CLERK_SECRET_KEY` empty): classic
  sessions only, `/auth/clerk-session` is not mounted. Re-enabling Clerk on
  Cloudflare is possible but the Node.js-middleware leg is still marked
  experimental by the OpenNext adapter — do that in a later milestone.
- For real usage, swap the free subdomains for a custom domain (e.g.
  `api.yourschool.com` + `app.yourschool.com`): shared registrable domain
  means you can set `COOKIE_SAMESITE=lax` again (still works with `none`,
  but lax is the tighter choice).

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
- 000011: attendance sessions and records
- 000012: assessments and results
- 000013: announcements

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
go run ./cmd/seed -role ADMIN -email your-admin@example.com -password '…'
go run ./cmd/seed -role TEACHER -email your-teacher@example.com -password '…'
```

These are operator-minted accounts meant for local testing only — pick your
own email/password; the repo ships no shared demo credentials (M34).

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

## Current M18 feature set

M18 is implemented in the backend and verified by tests. It includes:

- enrollment analytics by academic year and class utilization
- class attendance summaries with per-student breakdowns and aggregate rates
- class performance analysis across assessments and subjects
- student report cards combining attendance and subject-level performance
- route protection via the `report.view` permission and API gating in `cmd/server/main.go`

The implementation lives in:

- `backend/internal/reports/service.go`
- `backend/internal/reports/handlers.go`
- `backend/db/queries/reports.sql`
- `backend/cmd/server/main.go`

## Demo environment (M22)

The repo ships an idempotent demo-data seeder: `backend/cmd/demo-seed`. It
populates a coherent demo school — departments and positions, two teaching
staff with accounts (Ada Obi, Sarah Mensah), 12 Form 1 students across
Form 1A/1B with active enrollments, guardian links, teacher assignments,
the last 20 school days of attendance for each class-subject pairing, two
assessments per pairing with graded results, and a set of announcements
(published/draft/archived, school-wide/role/class targeted).

It is safe to re-run: every insert uses fixed UUIDs or the table's natural
unique key with `ON CONFLICT DO NOTHING`, and it never overwrites data
created through the API.

Run it against the schema-owning role (like migrations — it's an operator
tool, not the app; see `backend/cmd/demo-seed/main.go` for details):

```bash
# from backend/, with the local dev stack on host port 5433 (see above)
DATABASE_URL='postgres://schoolhub:schoolhub@localhost:5433/schoolhub?sslmode=disable' go run ./cmd/demo-seed
```

Demo login accounts (historical): admin@/ada@/teacher@/parent@/student@
schoolhub.test with the shared `ChangeMe123!` password were **removed in M34**
— the database no longer carries shared demo credentials. The seeder now
creates only structure data (people/staff/students/classes/enrollments,
attendance, assessments, announcements) with no login accounts; the tour
below signs in with your own operator account, and the milestone check
scripts use the gitignored `frontend/.check-credentials.json` QA personas
(see `frontend/scripts/lib/check-accounts.js`).

## Frontend scaffolding (M23)

The frontend has a shared foundation every screen builds on:

- **Login** (`/login`) — wired to the real session auth, with inline errors
  and role-based redirect (honors `?next=`).
- **Auth context** (`components/auth-provider.tsx`) — resolves `GET /me` on
  mount; the httpOnly session cookie stays the only credential.
- **ProtectedRoute** (`components/protected-route.tsx`) — declarative gating:
  redirects signed-out users to `/login?next=…`, shows an access-denied panel
  for users without the required role. Backend RBAC remains authoritative.
- **AppShell** (`components/app-shell.tsx`) — role-filtered sidebar, topbar
  with page title/actions slot, user chip, and sign-out; used by all workspace
  screens.
- **UI kit** (`components/ui/`) — `DataTable`, `SearchInput` (debounced),
  `Pagination`, `Modal`, and form field primitives for the admin screens of
  M24+.

A one-off browser verification script (`frontend/scripts/m23-browser-check.js`)
uses puppeteer-core against the local Chromium to exercise login, shell
rendering, role denial, and logout end-to-end.

## Staff management UI (M24)

The first admin screen built on the M23 scaffolding, backed by the M6 staff
domain:

- **Staff Directory** (`/staff`) — searchable, filterable (department,
  employment status, position category), paginated table with staff creation.
- **Staff profile** (`/staff/[id]`) — personal details, department/position
  assignment (full-replace), employment status, and login-account management
  (create account, replace roles) with per-action permission gating that
  mirrors the backend's route gates.
- **Departments & Positions** (`/staff/setup`) — reference-data management.

A browser verification script (`frontend/scripts/m24-browser-check.js`) uses
puppeteer-core to exercise the full flow — list, search, create, profile,
account + roles, reference data — end to end.

## Students & guardians UI (M25)

Built on the same scaffolding as staff management, with the shared UI kit
upgraded to the professional dashboard bar (zebra tables with responsive
column hiding, pill search, numbered pagination, pastel stat cards, initial
avatars):

- **Students** (`/students`) — directory with active/total/archived stat
  cards, search, status filter, pagination, and student enrollment.
- **Student profile** (`/students/[id]`) — personal details edit,
  archive/re-activate, and guardian relationship management (link existing
  guardians via search-and-pick; unlink with explicit confirmation).
- **Guardians** (`/guardians`, `/guardians/[id]`) — directory with
  linked-children counts, contact profile, edit, and linked-student cards.

A browser verification script (`frontend/scripts/m25-browser-check.js`) uses
puppeteer-core to exercise directory, search, enroll, profile, guardian
link/unlink, archive, and the guardians pages end to end.

## Academic structure UI (M26)

A tabbed admin workspace (`/academics`) for the M8 domain:

- **Academic Years / Terms** — tables with "Current" badges, per-row edit
  modals, and "Set current" actions (the backend enforces a single current
  year/term; setting another moves the flag).
- **Programmes / Subjects / Classes** — tables with create modals (subject
  type, class year/programme/level/capacity).
- **Class Curriculum** — pick a class, see its assigned subjects, assign new
  ones (only unassigned subjects offered), remove with one click.

Browser verification: `frontend/scripts/m26-browser-check.js` (puppeteer-core)
walks year/term/programme/subject creation, the current-flag move, and
curriculum assign/unassign end to end.

## Enrollment & teaching assignments UI (M27)

Two admin screens plus a profile section, covering the M9/M10 backends:

- **Enrollment register** (`/enrollments`) — date-sorted register with
  status/class/year pill filters, search, and pagination; per-row status
  transition select (ENROLLED → COMPLETED / TRANSFERRED / WITHDRAWN, gated on
  `class.update`); "+ Enroll student" modal with a debounced search-and-pick
  student list and a class dropdown filtered by the chosen academic year
  (mirroring the backend's class-year validation).
- **Teaching assignments** (`/teaching`) — teacher → class → subject coverage
  table; the create modal loads each class's curriculum so only subjects the
  class actually teaches can be assigned (same chain check as the backend),
  duplicates surface `ALREADY_ASSIGNED` inline, and one click removes an
  assignment.
- **Student profile enrollment history** — every past enrollment with class,
  year, dates, and notes; admins can enroll the student into the next
  academic year right from the profile.

Browser verification: `frontend/scripts/m27-browser-check.js` — 13 checks from
register rendering and filters through create/transition/history/assign/
duplicate-reject/remove. Note the DB enforces one active enrollment per
student/academic-year (a partial unique index over ENROLLED/COMPLETED/
TRANSFERRED) — test re-enrollments in a different year.

## How to continue from here

1. Make sure the database and backend are running.
2. Apply migrations if needed (`docker compose run --rm migrate up`).
3. Start the API and, if needed for local validation, the frontend.
4. Log in as a seeded admin or a matching role user and validate the report endpoints.
5. Seed the demo dataset (`backend/cmd/demo-seed`, see the M22 section above).
6. `cd frontend && npm run dev` for the UI at http://localhost:3000 and sign in
   with an operator account (schools ship their own — nothing is pre-seeded).
7. Compose and publish announcements from the /announcements screen (M30 —
   composer + directory with ALL/ROLE/CLASS targeting, priority, expiry,
   publish/archive/delete; school-wide publishes email staff best-effort).
8. Pull the Reports Center at /reports (M31 — enrollment KPIs + per-class
   utilization, class attendance analytics with term/date filters, class
   assessment performance, and printable student report cards).
9. Review the audit trail at /audit (M32 — filterable by action/resource
   type/date with per-entry metadata detail; admin only).
10. Enjoy consistent a11y everywhere: keyboard-navigable tabs on the
    multi-tab pages, focus-managed modals, always-linked labels.
11. Next up: M35 deployment.

Example flow:

```bash
cd /media/light/Grace/schoolhub_1
docker compose up -d postgres
POSTGRES_PORT=5433 docker compose up -d
docker compose run --rm migrate up
cd backend && go run ./cmd/server
```

Then, if you want a local UI check in a second terminal:

```bash
cd frontend && npm run dev
```

## Verification notes

The backend and all domain packages build and test successfully:

```bash
cd backend && go build ./... && go test ./...
```

The frontend is optional for milestone validation, and it is not treated as part
of milestone acceptance unless a milestone explicitly calls for UI work.

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
- M23 Frontend design system & admin shell
- M24 Admin: staff & reference data UI
- M25 Admin: students & guardians UI
- M26 Admin: academic structure UI
- M27 Admin: enrollment & teacher assignment UI
- M28 Attendance UI
- M29 Assessments & results UI
- Expert pass: professionalization (school landing page, demo text removed,
  optional Clerk hybrid-bridge auth, optional Resend email)

Next (frontend milestone expansion):

- Done: M30 Announcements composer UI — see the "Announcements composer
  (M30)" section below for what shipped and how it was verified.
- Done: M31 Reports UI — the /reports Reports Center (four tabs over M18's
  report endpoints, browser + API verified via frontend/scripts/reports-check.js).
- Done: M32 Audit log viewer UI — the /audit screen over M19's read
  endpoints (verified via frontend/scripts/audit-check.js, 18/18).
- Done: M33 Frontend polish & responsive pass — accessible tab/tablist
  widgets with keyboard selection on every tabbed page, modal focus
  management, always-linked form labels, navigation landmarks, a wrappable
  topbar (fixed the staff-directory mobile overflow), all verified by the
  mobile-viewport overflow + a11y sweep in frontend/scripts/polish-check.js
  (41/41).
- Done: M34 Master-admin access & real credentials — master-admin "Add
  admin" composer on /access (Clerk invite-only), forced first-login
  password-rotation gate, and the end of shared demo credentials in the DB
  (verified via frontend/scripts/access-admin-browser-check.js, 8/8 in
  classic mode; demo accounts removed in favor of gitignored QA personas).
  Inviter tracking + governance live-checked at 14/14 (API), 9/9 (UI), and
  4/4 (last-master guard staging). **Check-script modes**: browser legs that
  drive the #email form detect classic vs Clerk mode automatically — with
  Clerk keys active they print `SKIPPED … requires classic mode` and exit 0
  (API checks still run); switch the dev server to classic mode to execute
  them fully (frontend/scripts/lib/check-accounts.js).
- M35 Deployment
- M36 Documentation & portfolio prep

## Tech stack

Next.js · TypeScript · Tailwind CSS · Go · Gin · pgx · sqlc · PostgreSQL · Docker

