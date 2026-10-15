-- Reverse of the up migration: drop the new permission codes, the
-- MASTER_ADMIN role mapping, the blanket MASTER_ADMIN permissions, the
-- role itself and the CHECK widening.
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('admin.manage', 'account.block')
);

DELETE FROM permissions
WHERE code IN ('admin.manage', 'account.block');

-- Promotions granted in the up migration simply vanish with the role.
DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name = 'MASTER_ADMIN');

DELETE FROM roles
WHERE name = 'MASTER_ADMIN';

ALTER TABLE roles
    DROP CONSTRAINT roles_name_check;

ALTER TABLE roles
    ADD CONSTRAINT roles_name_check
    CHECK (name IN ('ADMIN', 'TEACHER', 'STAFF', 'PARENT', 'STUDENT'));
