# SchoolHub — Progress Log (remember.md)

Living document so any session (human or AI) can pick up exactly where the
project left off. Update it at the end of every milestone.

**Source of truth:** the milestone spec (M0–M24). Rules: one milestone at a
time, test before reporting, STOP after each milestone and wait for approval.

---

## Current status

| | |
|---|---|
| **Last completed milestone** | M5 — RBAC/permissions (2026-10-03) |
| **Next milestone** | M6 — People and staff (NOT started; waiting for approval) |
| **DB migration version** | 4 (clean) |
| **Backend tests** | `go test ./...` — rbac package: 12 tests passing |
| **Commits** | M0–M3 committed earlier; M4 `feat(auth)`, M5 `feat(rbac)`, `docs:` README+log |

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

## M5 — RBAC details (current milestone)

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

## Verified commands (M5)

```bash
cd backend && go build ./... && go vet ./... && go test ./...
docker compose up -d --build postgres backend   # rebuilt 2026-10-03 after DNS/port staleness
docker compose run --rm migrate up              # DB at version 4
cd frontend && npm run typecheck
```

E2E (curl against live server): unauthenticated 401 on /roles; teacher
403 on /roles & /permissions; admin can assign/revoke permissions on roles
(204, idempotent, reflected live in teacher's /me permissions); bad UUID →
400, unknown code → 404, missing body → 400; structured
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

## Next up: M6 — People and staff (waiting for approval)

Planned scope per spec §12 + §16: `staff`, `staff_positions`, `departments`
migrations; staff CRUD + search/filter/pagination (§25); activate/deactivate
status handling; link staff → user account + role assignment; first real
users of `RequirePermission` (staff.view/create/update/deactivate); audit
logging (§29) arrives as its own concern — confirm in M6 discussion whether
audit logging lands inside M6 or as M19.
