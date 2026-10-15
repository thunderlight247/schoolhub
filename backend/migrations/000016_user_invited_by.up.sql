-- M34 follow-up: inviter tracking + governance loosening.
--
-- The access-control redesign keeps one record of WHO invited an account:
-- every provisioned login (admins invited by MASTER_ADMIN, staff accounts
-- created by an admin, guardian accounts created by an admin or teacher)
-- stores its inviter on users.invited_by. With that link in place the
-- governance clauses become expressible:
--   * every inviter can list the accounts they invited (GET /my-invitees);
--   * every inviter can suspend/restore the accounts they invited, even
--     when the target's role would otherwise be above their own;
--   * MASTER_ADMIN can suspend/restore ANY admin (previously admins were
--     unblockable and managed purely by role revocation — the explicit
--     user requirement overrides that: blocking is suspend-restore, while
--     revoking ADMIN role remains the way to strip privileges).
--
-- Guardrails (enforced in the service layer): nobody blocks themselves,
-- and the system must never lose its last ACTIVE master admin — blocking
-- one is refused while it is the only ACTIVE MASTER_ADMIN. Existing
-- historical accounts keep invited_by NULL — no back-dating.

-- Teachers invite parents (user requirement): a teachers-only permission
-- code covering guardian creation AND parent-account provisioning, so the
-- M7 guardians routes can be split off the blunt student.create/staff.update
-- gates without letting teachers mint students or touch staff.
INSERT INTO permissions (code, description)
VALUES
    ('guardian.create', 'Invite a guardian/link and provision the parent login account. Held by TEACHER alongside the admin roles.')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE p.code = 'guardian.create' AND r.name IN ('ADMIN', 'MASTER_ADMIN', 'TEACHER')
ON CONFLICT DO NOTHING;

ALTER TABLE users
    ADD COLUMN invited_by UUID NULL REFERENCES users (id);

CREATE INDEX idx_users_invited_by ON users (invited_by);
