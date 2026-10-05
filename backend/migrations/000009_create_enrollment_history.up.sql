-- Enrollment history (milestone M9).
--
-- This table is the canonical place for a student's academic membership. The
-- student record itself remains class-agnostic (M7 deliberately avoids a
-- permanent class field); enrollment history captures which class a student was
-- in during a given academic year and how that membership changed over time.

CREATE TABLE enrollments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id        UUID NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
    class_id         UUID NOT NULL REFERENCES classes (id) ON DELETE RESTRICT,
    status           TEXT NOT NULL DEFAULT 'ENROLLED' CHECK (status IN ('ENROLLED', 'COMPLETED', 'TRANSFERRED', 'WITHDRAWN')),
    enrollment_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    exit_date        DATE,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_enrollments_dates CHECK (exit_date IS NULL OR exit_date >= enrollment_date)
);

CREATE INDEX idx_enrollments_student_id ON enrollments (student_id);
CREATE INDEX idx_enrollments_academic_year_id ON enrollments (academic_year_id);
CREATE INDEX idx_enrollments_class_id ON enrollments (class_id);
CREATE INDEX idx_enrollments_status ON enrollments (status);

CREATE TRIGGER trg_enrollments_set_updated_at
    BEFORE UPDATE ON enrollments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- One active enrollment per student within a single academic year. This allows
-- historical records to stay intact while preventing duplicate live class
-- memberships for the same year.
CREATE UNIQUE INDEX idx_enrollments_unique_active_year
    ON enrollments (student_id, academic_year_id)
    WHERE status IN ('ENROLLED', 'COMPLETED', 'TRANSFERRED');
