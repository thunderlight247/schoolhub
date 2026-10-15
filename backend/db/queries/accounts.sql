-- Admin-account management (expert pass follow-up). Queries behind the
-- admin.manage (list/grant/revoke admins) and account.block (suspend/
-- restore any login account) endpoints. Kept in their own query file —
-- like invitation_tokens.sql, attention here is narrow and permission-
-- sensitive, so the SQL lives apart from the big domain files.

-- name: ListAdmins :many
-- Everyone currently holding the ADMIN or MASTER_ADMIN role, with account
-- status so the master admin can spot suspended admins at a glance.
SELECT DISTINCT
    u.id            AS user_id,
    u.email,
    u.status,
    u.last_login_at,
    p.first_name,
    p.last_name,
    COALESCE((SELECT r2.name
     FROM user_roles ur2
     JOIN roles r2 ON r2.id = ur2.role_id
     WHERE ur2.user_id = u.id AND r2.name = 'MASTER_ADMIN'
     LIMIT 1), '') AS is_master
FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r       ON r.id = ur.role_id
JOIN people p      ON p.id = u.person_id
WHERE r.name IN ('ADMIN', 'MASTER_ADMIN')
ORDER BY p.last_name, p.first_name;

-- name: ListAccountsWithRole :many
-- Account directory for the access-control screen: everyone holding a
-- chosen role with their account status. role_name is passed as TEXT
-- rather than a uuid to keep the frontend simple (roles are a fixed set).
SELECT
    u.id            AS user_id,
    u.email,
    u.status,
    u.last_login_at,
    p.id            AS person_id,
    p.first_name,
    p.last_name
FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r       ON r.id = ur.role_id
JOIN people p      ON p.id = u.person_id
WHERE r.name = sqlc.arg(role_name)
ORDER BY p.last_name, p.first_name;

-- name: CountAdmins :one
-- Active holders of the ADMIN role — the guard against revoking the last
-- one. MASTER_ADMIN holders count: a master admin may not *revoke their
-- own* admin powers anyway (service check), but any actively-serving
-- admin keeps the system manageable.
SELECT count(DISTINCT ur.user_id) AS active_admins
FROM user_roles ur
JOIN roles r   ON r.id = ur.role_id
JOIN users u   ON u.id = ur.user_id
WHERE r.name IN ('ADMIN', 'MASTER_ADMIN') AND u.status = 'ACTIVE';

-- name: CountByRole :one
-- Active role holders in a given role, same "don't lock the system out"
-- guard when blocking accounts.
SELECT count(*) AS active_holders
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
JOIN users u ON u.id = ur.user_id
WHERE r.name = sqlc.arg(role_name) AND u.status = 'ACTIVE';
