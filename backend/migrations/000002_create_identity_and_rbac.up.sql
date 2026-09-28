-- Identity model (section 9): Person -> Student/Staff/Guardian -> User -> Role -> Permissions.
-- This migration creates the "spine" of that model: people, user accounts,
-- and role-based access control. The Student/Staff/Guardian tables that hang
-- off `people` are created in later milestones (M6/M7), each in its own
-- migration, per section 8's instruction not to put everything in one
-- giant migration.

-- ---------------------------------------------------------------------------
-- people: the single source of truth for a human's identity data. A person
-- may go on to be a student, staff member, and/or guardian (a staff member
-- can also be a parent of a student, for example) — those roles are modeled
-- as separate tables referencing people.id, not as flags on this table.
-- ---------------------------------------------------------------------------
CREATE TABLE people (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name     TEXT NOT NULL,
    last_name      TEXT NOT NULL,
    other_names    TEXT,
    date_of_birth  DATE,
    gender         TEXT CHECK (gender IN ('MALE', 'FEMALE', 'OTHER')),
    phone          TEXT,
    email          TEXT,
    address        TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Supports name search/filtering for admin staff & student search screens.
CREATE INDEX idx_people_name ON people (last_name, first_name);

CREATE TRIGGER trg_people_set_updated_at
    BEFORE UPDATE ON people
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- users: login credentials for a person. One user account per person
-- (enforced by the UNIQUE constraint on person_id) — a person does not get
-- a separate login for each role they hold; roles are assigned via
-- user_roles instead.
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id      UUID NOT NULL UNIQUE REFERENCES people (id) ON DELETE RESTRICT,
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'ACTIVE'
                       CHECK (status IN ('ACTIVE', 'SUSPENDED', 'INACTIVE')),
    last_login_at  TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_person_id ON users (person_id);

CREATE TRIGGER trg_users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ON DELETE RESTRICT on users.person_id: a person record must not be
-- deletable while a user account still references it. In practice people
-- are never hard-deleted anyway (see section 7/12/14 on preserving
-- historical records) — this constraint just makes that explicit at the
-- database level rather than relying on application discipline alone.

-- ---------------------------------------------------------------------------
-- roles: the five system roles from section 10. Kept as a real table
-- (rather than a hardcoded application constant) so role_permissions can be
-- data-driven, even though the set of roles itself is fixed by the CHECK
-- constraint for this stage of the project.
-- ---------------------------------------------------------------------------
CREATE TABLE roles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL UNIQUE
                     CHECK (name IN ('ADMIN', 'TEACHER', 'STAFF', 'PARENT', 'STUDENT')),
    description  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- permissions: the permission codes from section 11 (e.g. "student.view").
-- Seeded below with the exact list from the spec so M5 (RBAC) has real data
-- to assign to roles rather than having to author it from scratch.
-- ---------------------------------------------------------------------------
CREATE TABLE permissions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         TEXT NOT NULL UNIQUE,
    description  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- role_permissions / user_roles: pure join tables. Unlike the historical
-- domain tables (enrollments, attendance, results, ...), these just express
-- a current assignment with no history worth preserving, so ON DELETE
-- CASCADE is appropriate here (see section 7's "not blindly" — this is a
-- deliberate exception, not the default).
-- ---------------------------------------------------------------------------
CREATE TABLE role_permissions (
    role_id        UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id  UUID NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id  UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- ---------------------------------------------------------------------------
-- Seed data: the roles and permissions are fixed reference/lookup data
-- defined by the spec itself (sections 10 and 11) — not fictional school
-- data, so seeding it here (rather than in M22's demo dataset) is
-- appropriate. ON CONFLICT DO NOTHING keeps this migration re-runnable.
--
-- Deliberately NOT seeded here: role_permissions (which permissions each
-- role actually gets). Deciding that mapping is RBAC business logic and
-- belongs to M5, not to schema setup.
-- ---------------------------------------------------------------------------
INSERT INTO roles (name, description) VALUES
    ('ADMIN',   'Full administrative access to SchoolHub.'),
    ('TEACHER', 'Teaching staff with access to their assigned classes and subjects.'),
    ('STAFF',   'Non-teaching staff with limited administrative access.'),
    ('PARENT',  'Guardian/parent portal access, scoped to their own children.'),
    ('STUDENT', 'Student portal access, scoped to their own records.')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (code, description) VALUES
    ('student.view',        'View student records.'),
    ('student.create',      'Register new students.'),
    ('student.update',      'Edit student records.'),
    ('student.archive',     'Archive/deactivate students.'),
    ('staff.view',          'View staff records.'),
    ('staff.create',        'Add new staff.'),
    ('staff.update',        'Edit staff records.'),
    ('staff.deactivate',    'Deactivate staff.'),
    ('class.view',          'View classes.'),
    ('class.create',        'Create classes.'),
    ('class.update',        'Edit classes.'),
    ('attendance.view',     'View attendance records.'),
    ('attendance.record',   'Record attendance.'),
    ('attendance.update',   'Edit attendance records.'),
    ('assessment.view',     'View assessments.'),
    ('assessment.create',   'Create assessments.'),
    ('assessment.update',   'Edit assessments.'),
    ('result.view',         'View results.'),
    ('result.create',       'Enter results.'),
    ('result.update',       'Edit results.'),
    ('report.view',         'View reports.'),
    ('report.export',       'Export reports.'),
    ('announcement.view',   'View announcements.'),
    ('announcement.create', 'Create announcements.'),
    ('announcement.update', 'Edit announcements.'),
    ('audit.view',          'View audit logs.')
ON CONFLICT (code) DO NOTHING;
