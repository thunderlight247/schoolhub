# SchoolHub — Progress Log (remember.md)

Living document so any session (human or AI) can pick up exactly where the
project left off. Update it at the end of every milestone.

**Source of truth:** the milestone spec (M0–M24). Rules: one milestone at a
time, test before reporting, STOP after each milestone and wait for approval.

---

## Current status

| | |
|---|---|
| **Last completed milestone** | M6 — People and staff (implemented 2026-10-04; UNCOMMITTED, ready for review) |
| **Next step** | Review M6 work → commit; then M7 per spec (waiting for approval) |
| **DB migration version** | 6 (000005 departments/staff_positions/staff, 000006 audit_logs — applied to dev DB) |
| **Backend tests** | `go test ./...` — rbac: 12 passing; staff: 21 passing (pagination + handler suite) |
| **Commits** | M0–M3 committed earlier; M4 `feat(auth)`, M5 `feat(rbac)`, `docs:` README+log. M6 exists only as working-tree changes so far. |

## Milestones completed

- **M0 — Project initialization.** Repo structure (frontend/ + backend/ +
  docs/), Docker Compose (postgres + backend + migrate/sqlc tool services),
  Next.js scaffold, Go scaffold, health-check endpoint, env handling,
  README.
- **M1 — Architecture.** Modular monolith: `backend/internal/<domain>`,
  handler → service → sqlc repository layering. Placeholder `doc.go` files
  for domain packages not yet built.
- **M2 — Migrations 000001–000002.** `pgcrypto` + `set_updated_at()` trigger;
  identity spine: `people`, `users`, `roles`, `permissions`,
  `role_permissions`, `user_roles` — seeded with the 5 roles and the 26
  permission codes from the spec.
- **M3 — sqlc.** `backend/sqlc.yaml`; queries + generated code per table
  (`db/queries/*.sql` → `db/generated/`). Generated code is committed.
- **M4 — Authentication (commit `feat(auth)`).** Login/logout/`/me` with
  httpOnly session cookie (`schoolhub_session`), Argon2id password hashing,
  opaque 256-bit tokens stored as SHA-256 hashes (migration 000003:
  `sessions`, `password_reset_tokens`, `invitation_tokens` — the token
  tables are schema-only until their features arrive). Suspended/inactive
  accounts cannot log in; `internal/apierror` package for structured errors;
  temporary `cmd/seed` command to create users with a role.
- **M5 — RBAC/permissions (commit `feat(rbac)`).** See details below.

## M5 — RBAC details (previous milestone)

- `internal/rbac/service.go` — `HasPermission(userID, code)` via
  `UserHasPermission` query; `HasRole` via `ListRolesForUser`.
- `internal/rbac/middleware.go` — `RequirePermission(code)` (the standard
  gate for M6+ domain routes) and `RequireRole(name)` (reserved for RBAC
  management itself; no permission code covers "manage permissions").
- `internal/rbac/handlers.go` — admin endpoints: list roles/permissions,
  list/assign/revoke a role's permissions (assign & revoke idempotent).
  Mounted in `cmd/server/main.go` behind `RequireAuth` + `RequireRole(ADMIN)`.
- Migration 000004 — default role→permission mapping (ADMIN=all 26;
  TEACHER=13 class-scoped; STAFF=7 limited-admin; PARENT/STUDENT=3
  view-only each). Row-level scoping ("parents see only their children")
  is deliberately deferred to M7/M11/M12 queries — RBAC only knows codes.
- **Auth API extension (approved):** `POST /auth/login` and `GET /me` now
  return `roles: []` and `permissions: []` (effective codes through
  user_roles → role_permissions) for the authenticated user. Informational
  for frontend UX; enforcement stays server-side.
- `auth.WithUser(c, user)` helper added for tests only.
- Tests: `internal/rbac/{middleware,service}_test.go` — stub `db.Querier`
  (embeds interface, unexpected calls panic). Cover 401/403/500 fail-closed
  paths and pass-through for both middlewares, plus service delegation.

## M6 — People and staff details (current milestone, uncommitted)

- **Decision recorded:** audit logging (§29) lands INSIDE M6 (write path
  only). Migration 000006 `audit_logs` + `internal/audit` package with
  `Record(ctx, txQuerier, Entry)` — every mutation writes its audit row in
  the SAME transaction, so an action either fully happens with its proof or
  not at all. Authorized read/report endpoints (`audit.view`) stay in M19.
- Migrations 000005/000006 per the file-header comments (nullable
  department/position FKs, RESTRICT deletes, employment_status CHECK,
  append-only audit table, indexes for M19 screens).
- `internal/staff/service.go` — `*pgxpool.Pool` + concrete `*db.Queries`
  (WithTx needs the concrete type; transactional paths get E2E coverage,
  handler paths get unit coverage). Methods: CreateStaff (person + staff +
  audit), GetStaff (+ optional account w/ roles), ListStaff (filtered +
  counted, `NormalizePagination` clamps page∈[1,∞), size∈[1,100]),
  UpdateStaff (PATCH person fields, nil = unchanged), SetAssignment
  (full-replace department/position/hired_date, nil = clear),
  SetEmploymentStatus (validates vs CHECK set, audits old→new),
  CreateAccount (Argon2id via auth.HashPassword; email/account-exists
  checks + race-safe unique-violation mapping; roles by name),
  SetAccountRoles (wholesale replace via ClearUserRoles; empty list =
  revoke all), CreateDepartment/CreateStaffPosition (+ audit).
- `internal/staff/handlers.go` + `reference_handlers.go` — handlers depend
  on the `ServiceAPI` interface (stub-able in tests); sentinel-error →
  HTTP mapping table (404/409/400/500); actor pulled from auth.CurrentUser
  for audit attribution. GETs return `{staff|departments|staff_positions:
  [...]}`; list adds `pagination{page,page_size,total,total_pages}`.
- Routes (main.go), all behind RequireAuth, gated by RequirePermission:
  staff.view → GET /staff, /staff/:id, /departments, /staff-positions;
  staff.create → POST /staff; staff.update → PATCH /staff/:id,
  PUT /staff/:id/assignment, POST /staff/:id/account,
  PUT /staff/:id/account/roles, POST /departments,
  POST /staff-positions; staff.deactivate → PUT /staff/:id/status.
  (Reference-data creates gate on staff.update — no dedicated code exists
  in the §11 set, and only ADMIN holds staff.update by default.)
- Tests: `staff/service_test.go` (pagination clamping; status/category
  sets vs migration CHECKs) + `staff/handlers_test.go` (stub ServiceAPI,
  embedded-interface panic on unexpected calls — rbac pattern). Cover
  filter parsing/clamping, validation 400s, sentinel→HTTP mapping (409
  conflicts, 404s), success payloads, actor attribution.
- E2E verified against rebuilt container: login → create dept/position →
  create staff → search+filter list → PATCH phone → ON_LEAVE → create
  account (TEACHER) → duplicate account 409 → no-cookie 401. Audit log
  shows all six expected rows with correct metadata.
- Known scope note: no account SUSPEND/ACTIVATE endpoint yet (users.status
  stays ADMIN/seed-managed until the users milestone); PATCH cannot clear
  a field to NULL (nil = unchanged) — flag if the frontend needs it.

## Key decisions made so far

- Opaque session tokens (not JWTs), hashed with SHA-256; Argon2id only for
  passwords (its slowness is wasted on high-entropy tokens).
- Coarse auth errors: invalid email vs wrong password are indistinguishable
  (no account enumeration).
- `role_permissions`/`user_roles` use ON DELETE CASCADE (current-state join
  tables, no history worth keeping); domain tables will use RESTRICT.
- Five roles fixed by CHECK constraint for now; permission→role mapping is
  editable data via the admin endpoints.
- `RequireRole(ADMIN)` only for RBAC management; everything else must use
  `RequirePermission(code)`.
- Frontend not containerized in dev; migrate/sqlc run via Docker tool
  services, no local CLI installs needed.

## Demo/dev accounts (fictional, DEV ONLY)

| Role | Email | Password |
|---|---|---|
| ADMIN | admin@schoolhub.test | ChangeMe123! |
| TEACHER | teacher@schoolhub.test | ChangeMe123! |

Created with `cd backend && go run ./cmd/seed -role <ROLE> -email <E> -password <P>`.

## Verified commands (M6)

```bash
cd backend && go build ./... && go vet ./... && go test ./...
docker compose run --rm migrate up              # DB at version 6
docker compose up -d --build backend            # rebuilt 2026-10-04 with M6 code
cd frontend && npm run typecheck
```

E2E M6 (curl against live server): full staff lifecycle — admin login →
create department/position → create staff (joins resolve names) →
search+filter list (pagination metadata correct) → PATCH phone → ON_LEAVE
status → create TEACHER account → duplicate account 409 → no-cookie 401.
Audit table shows DEPARTMENT_CREATED, POSITION_CREATED, STAFF_CREATED,
STAFF_UPDATED {fields:[phone]}, STAFF_STATUS_CHANGED {old,new},
USER_CREATED {email,roles,staff_id} — all in order with correct metadata.
M5's E2E results (below) still hold: unauthenticated 401 on /roles;
teacher 403 on /roles & /permissions; admin assign/revoke permissions
(204, idempotent); bad UUID → 400, unknown code → 404; structured
`{"error":{code,message}}` shape; logout invalidates session.

## Known issues / deferred

- Session-expiry cleanup (`DeleteExpiredSessions`) has no scheduler —
  expired sessions are rejected at validation time anyway (deliberate).
- `ValidateSession` does 2 extra small queries per request to resolve
  roles/permissions — fine at this scale; revisit only if it shows.
- Split-domain production deployment will need `COOKIE_DOMAIN`/CORS
  revisited (noted in M4 report).
- docs/ directory still empty; full documentation set is milestone M24
  (README carries the essentials until then).
- `backend/cookies.txt` is a curl artifact — gitignored, safe to delete.

## Next up: M7 (per spec) — waiting for approval

M6 is implemented and E2E-verified but UNCOMMITTED — review the working
tree, then commit as `feat(staff)` (migrations 000005–000006, sqlc code,
internal/audit, internal/staff, main.go wiring, remember.md update).

Open M6 items deliberately deferred (flag if wanted in M7 scope):
- Account SUSPEND/ACTIVATE endpoint (users.status) — employment status and
  login status are independent; the account-status toggle hasn't a home
  milestone yet (candidate: users milestone).
- PATCH cannot clear optional person fields to NULL (nil = unchanged).
- Audit read/report endpoints (`audit.view`) are M19 by design.

Planned scope for M7 per spec: students domain (§13) — students table +
guardians/enrollment groundwork, following the same
migration → sqlc → service → handlers → gated-routes pattern proven in M6.
