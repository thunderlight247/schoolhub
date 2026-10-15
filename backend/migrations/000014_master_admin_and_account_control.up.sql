-- Master-admin / account-access control (expert-pass follow-up):
--
-- Three related needs, one migration:
--   1. A MASTER_ADMIN role above ADMIN: the only role allowed to grant or
--      revoke the ADMIN role itself (+ the admin.manage permission). Plain
--      ADMINs keep day-to-day power (staff/students/classes/...) but
--      cannot manufacture more admins, closing the "admin promotes
--      friends" hole without rewriting the whole permission model.
--   2. Two new permission codes: admin.manage governs *admin* access
--      (list admins, grant/revoke the ADMIN role), account.block governs
--      suspending/unsuspending any parent/teacher account that already
--      exists. Both attach to roles through the existing role_permissions
--      join — no new tables.
--   3. Seed data: every user already holding ADMIN gets MASTER_ADMIN too,
--      and MASTER_ADMIN receives every permission (mirroring how
--      migration 000004 gives plain ADMIN the full set). In a real school
--      there is exactly one master admin to start with; upgrading
--      existing admins keeps the system consistent (somebody must be
--      able to manage admins on day one).
--
-- Deliberately NOT in this migration: changes to the users.status CHECK
-- (blocking reuses SUSPENDED — one status value, clearer semantics), and
-- no new tables at all. INVITE flow for parents keeps using
-- invitation_tokens (migration 000003) + Clerk sign-up.

ALTER TABLE roles
    DROP CONSTRAINT roles_name_check;

ALTER TABLE roles
    ADD CONSTRAINT roles_name_check
    CHECK (name IN (
        'ADMIN',
        'MASTER_ADMIN',
        'TEACHER',
        'STAFF',
        'PARENT',
        'STUDENT'
    ));

INSERT INTO roles (name, description)
VALUES (
    'MASTER_ADMIN',
    'Super administrator: can also grant or revoke other admins'' access. Holds every permission.'
)
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (code, description)
VALUES
    ('admin.manage',  'Manage administrator access: list admins, grant or revoke the ADMIN role. Held only by MASTER_ADMIN.'),
    ('account.block', 'Suspend or restore any parent/teacher/staff login account, revoking access even while signed in.')
ON CONFLICT (code) DO NOTHING;

-- MASTER_ADMIN receives every permission (including other roles'
-- codes). Same blanket rule as 000004 gives plain ADMIN.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'MASTER_ADMIN'
ON CONFLICT DO NOTHING;

-- Plain ADMIN keeps everything GRANTED so far (the blanket SELECT from
-- 000004) plus the new account.block for parent/teacher access suspension,
-- but deliberately NOT admin.manage.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code NOT IN ('admin.manage')
WHERE r.name = 'ADMIN'
ON CONFLICT DO NOTHING;

-- Any user already holding ADMIN gains MASTER_ADMIN too — keeps the
-- bootstrap invariant "someone can always manage admins". This is how a
-- typical school is seeded anyway one master admin plus sub-admins, so
-- doing it here is safe (further MASTER_ADMIN grant/revoke is then
-- master-only.)
INSERT INTO user_roles (user_id, role_id)
SELECT ur.user_id, ma.id
FROM user_roles ur
JOIN roles        a  ON a.id = ur.role_id
JOIN roles        ma ON ma.name = 'MASTER_ADMIN'
WHERE a.name = 'ADMIN'
ON CONFLICT DO NOTHING;
