-- Teacher assignments (milestone M10).
--
-- Maps which teacher teaches which subject in which class. Structure mirrors
-- class_subjects (M8): a thin mapping table, with the class -> academic_year
-- relationship (000008) providing the temporal scoping — a new school year
-- means new classes and therefore new assignments.
--
-- Deliberate choices:
--   * teacher_id references staff, not users: a teaching assignment belongs
--     to the employment record (M6), so it must survive login/account
--     changes and cannot exist for a person without a staff record.
--   * ON DELETE CASCADE on all three FKs: an assignment is derived coverage
--     data — if the teacher, class, or subject disappears, the assignment
--     has no meaning left to protect and must not block (RESTRICT) the
--     parent row. Same behavior as class_subjects in 000008.
--   * UNIQUE (teacher_id, class_id, subject_id): one assignment per teacher
--     per class-subject pairing. A different teacher may hold the same
--     class-subject (co-teaching), and one teacher may hold many subjects.

CREATE TABLE teacher_assignments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id  UUID NOT NULL REFERENCES staff (id) ON DELETE CASCADE,
    class_id    UUID NOT NULL REFERENCES classes (id) ON DELETE CASCADE,
    subject_id  UUID NOT NULL REFERENCES subjects (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (teacher_id, class_id, subject_id)
);

CREATE INDEX idx_teacher_assignments_teacher_id ON teacher_assignments (teacher_id);
CREATE INDEX idx_teacher_assignments_class_id   ON teacher_assignments (class_id);
CREATE INDEX idx_teacher_assignments_subject_id ON teacher_assignments (subject_id);

CREATE TRIGGER trg_teacher_assignments_set_updated_at
    BEFORE UPDATE ON teacher_assignments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
