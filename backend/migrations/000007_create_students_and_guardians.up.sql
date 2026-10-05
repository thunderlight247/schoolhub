-- Students and guardians (sections 14-15, milestone M7).
--
-- Deliberate choices, consistent with the staff domain's precedent
-- (migration 000005):
--   * Guardians live in this same migration and in the same Go package
--     (internal/students) as students, not a separate internal/guardians
--     package, even though M0 originally stubbed internal/guardians/doc.go
--     on its own. The two are too tightly coupled here (every guardian
--     mutation that matters in this milestone happens in the context of
--     linking/unlinking a student) to split across packages without
--     awkward cross-package transaction composition — the same reasoning
--     M6 used to fold departments and staff_positions into internal/staff
--     rather than giving them their own packages.
--   * students.status is a 2-state CHECK (ACTIVE, ARCHIVED), not staff's
--     4-state employment_status — section 14 only ever says "archive/
--     deactivate students", one action, not several distinct states the
--     way section 12 explicitly lists for staff.
--   * No class/programme/academic_year reference anywhere on students.
--     Section 14 is explicit: "Student records should not simply contain
--     one permanent class field. Use enrollment history" — that history
--     lives in the enrollments table, which is M9's job, not this one.
--   * student_guardians.relationship is free TEXT, not a CHECK-constrained
--     enum — section 15 never gives a fixed relationship list the way
--     section 12 gives a fixed position-category list, so constraining it
--     would be inventing a rule the spec doesn't state.
--   * No "is_primary guardian" flag: not asked for in section 15, and
--     nothing in this milestone needs to resolve guardian precedence — a
--     small addition to reach for later if a real need shows up, not
--     speculative infrastructure now (section 39).
--   * ON DELETE CASCADE on student_guardians' two FKs: this is a pure
--     association (same reasoning as role_permissions/user_roles in
--     migration 000002) — the relationship record has no standalone value
--     once either side is gone, and neither students nor guardians are
--     ever hard-deleted in practice (archived instead), so the cascade
--     never actually fires outside of tests.
--   * ON DELETE RESTRICT on students.person_id / guardians.person_id,
--     matching every other person-backed table (staff, users): a person
--     record must outlive the student/guardian row that references it.

CREATE TABLE students (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id      UUID NOT NULL UNIQUE REFERENCES people (id) ON DELETE RESTRICT,
    student_number TEXT NOT NULL UNIQUE,
    admission_date DATE,
    status         TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_students_status ON students (status);

CREATE TRIGGER trg_students_set_updated_at
    BEFORE UPDATE ON students
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE guardians (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id   UUID NOT NULL UNIQUE REFERENCES people (id) ON DELETE RESTRICT,
    occupation  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_guardians_set_updated_at
    BEFORE UPDATE ON guardians
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE student_guardians (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id   UUID NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    guardian_id  UUID NOT NULL REFERENCES guardians (id) ON DELETE CASCADE,
    relationship TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (student_id, guardian_id)
);

-- UNIQUE(student_id, guardian_id) above already indexes student_id first;
-- this adds the reverse direction ("which students does this guardian
-- have") which that composite index doesn't serve.
CREATE INDEX idx_student_guardians_guardian_id ON student_guardians (guardian_id);
