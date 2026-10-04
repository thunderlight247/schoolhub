-- Staff management domain (section 12, milestone M6).
--
-- Staff model per section 9's identity rule: a STAFF row is an employment
-- record hanging off a person, never a copy of person data. One staff row
-- per person (UNIQUE person_id) — a person is employed by the school once.
--
-- Deliberate choices:
--   * department_id / position_id are nullable: "Assign department" and
--     "Assign position" (section 12) are actions an admin takes on an
--     existing staff record, so a record can legitimately exist before
--     either is assigned. ON DELETE RESTRICT keeps a department/position
--     from disappearing out from under historical employment records
--     (section 7) — in practice these lookups are never hard-deleted.
--   * employment_status CHECK mirrors section 12's list (ACTIVE, INACTIVE,
--     SUSPENDED, ON_LEAVE) and is separate from users.status (login
--     account state) — deactivating a staff member's employment does not
--     automatically disable their login; that's an explicit admin action
--     on the account.
--   * staff_number is the human-readable identifier schools actually use
--     day to day ("find staff by number"), unique and NOT NULL.
--   * The category list (Teaching, Administration, ICT, Accounts,
--     Management, Support from section 12) lives on staff_positions: a
--     position belongs to a category, and staff inherit it through their
--     position. Changing the list later is a small migration, not a
--     lookup-table detour (section 39).

CREATE TABLE departments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_departments_set_updated_at
    BEFORE UPDATE ON departments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE staff_positions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL UNIQUE,
    category    TEXT NOT NULL CHECK (category IN
                    ('TEACHING', 'ADMINISTRATION', 'ICT', 'ACCOUNTS',
                     'MANAGEMENT', 'SUPPORT')),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_staff_positions_set_updated_at
    BEFORE UPDATE ON staff_positions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE staff (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id         UUID NOT NULL UNIQUE REFERENCES people (id) ON DELETE RESTRICT,
    staff_number      TEXT NOT NULL UNIQUE,
    department_id     UUID REFERENCES departments (id) ON DELETE RESTRICT,
    position_id       UUID REFERENCES staff_positions (id) ON DELETE RESTRICT,
    employment_status TEXT NOT NULL DEFAULT 'ACTIVE'
                          CHECK (employment_status IN
                              ('ACTIVE', 'INACTIVE', 'SUSPENDED', 'ON_LEAVE')),
    hired_date        DATE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_staff_department_id     ON staff (department_id);
CREATE INDEX idx_staff_position_id       ON staff (position_id);
CREATE INDEX idx_staff_employment_status ON staff (employment_status);

CREATE TRIGGER trg_staff_set_updated_at
    BEFORE UPDATE ON staff
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
