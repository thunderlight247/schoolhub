-- Default role -> permission mapping (section 11). This is a judgment call,
-- not something stated explicitly in the spec, so it's documented here
-- rather than silently assumed:
--
--   ADMIN   — every permission.
--   TEACHER — their classes/subjects: students, attendance, assessments,
--             results, class-scoped announcements. No staff management,
--             no cross-class student editing.
--   STAFF   — non-teaching admin support: student records, viewing staff,
--             viewing classes, reports, announcements. Deliberately not
--             full ADMIN (no staff.create/update/deactivate, no RBAC
--             management) — "non-teaching staff with LIMITED
--             administrative access" (section 10).
--   PARENT  — view-only: attendance, results, announcements. Narrowing
--             this to "their own children's records" is a data-scoping
--             concern for the queries built in M7/M11/M12, not something
--             RBAC (which only knows permission codes, not row ownership)
--             can express — flagging that boundary explicitly.
--   STUDENT — view-only, same scoping caveat as PARENT.
--
-- Pure data, no schema change, so this is its own migration rather than
-- folded into 000002 — the mapping is more likely to be revisited later
-- (and partly superseded by the admin RBAC-management endpoints this
-- milestone also adds) than the identity schema itself is.

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'ADMIN'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'student.view',
    'class.view',
    'attendance.view', 'attendance.record', 'attendance.update',
    'assessment.view', 'assessment.create', 'assessment.update',
    'result.view', 'result.create', 'result.update',
    'announcement.view', 'announcement.create'
)
WHERE r.name = 'TEACHER'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'student.view', 'student.create', 'student.update',
    'staff.view',
    'class.view',
    'report.view',
    'announcement.view'
)
WHERE r.name = 'STAFF'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'attendance.view',
    'result.view',
    'announcement.view'
)
WHERE r.name = 'PARENT'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'attendance.view',
    'result.view',
    'announcement.view'
)
WHERE r.name = 'STUDENT'
ON CONFLICT DO NOTHING;
