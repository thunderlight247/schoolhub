-- Academic structure (milestone M8).
--
-- This migration establishes the institutional academic scaffolding needed
-- before enrollment (M9) and teacher assignments (M10):
--   * academic_years: institutional school years with start/end bounds and
--     at most one active/current year.
--   * terms: academic terms/semesters linked to an academic year, with at most
--     one active/current term across the school.
--   * programmes: curriculum streams/tracks (e.g. Science, Arts, Business).
--   * subjects: catalog of academic subjects (Core or Elective), optionally
--     associated with a department (from M6).
--   * classes: cohorts/classrooms bound to an academic year and optional
--     programme, with unique class names within the academic year.
--   * class_subjects: many-to-many relationship mapping which subjects are
--     taught in a particular class.

CREATE TABLE academic_years (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    is_current  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_academic_years_dates CHECK (end_date >= start_date)
);

CREATE TRIGGER trg_academic_years_set_updated_at
    BEFORE UPDATE ON academic_years
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Enforce that at most one academic year can be marked as current at any time.
CREATE UNIQUE INDEX idx_academic_years_current
    ON academic_years (is_current)
    WHERE is_current = true;

CREATE TABLE terms (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    academic_year_id  UUID NOT NULL REFERENCES academic_years (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    start_date        DATE NOT NULL,
    end_date          DATE NOT NULL,
    is_current        BOOLEAN NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_terms_dates CHECK (end_date >= start_date),
    UNIQUE (academic_year_id, name)
);

CREATE INDEX idx_terms_academic_year ON terms (academic_year_id);

CREATE TRIGGER trg_terms_set_updated_at
    BEFORE UPDATE ON terms
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Enforce that at most one term can be marked as current at any time.
CREATE UNIQUE INDEX idx_terms_current
    ON terms (is_current)
    WHERE is_current = true;

CREATE TABLE programmes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL UNIQUE,
    description  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_programmes_set_updated_at
    BEFORE UPDATE ON programmes
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE subjects (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code           TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    description    TEXT,
    type           TEXT NOT NULL DEFAULT 'CORE' CHECK (type IN ('CORE', 'ELECTIVE')),
    department_id  UUID REFERENCES departments (id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subjects_department ON subjects (department_id);

CREATE TRIGGER trg_subjects_set_updated_at
    BEFORE UPDATE ON subjects
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE classes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    academic_year_id  UUID NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
    programme_id      UUID REFERENCES programmes (id) ON DELETE RESTRICT,
    name              TEXT NOT NULL,
    level             TEXT,
    capacity          INTEGER CHECK (capacity IS NULL OR capacity > 0),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (academic_year_id, name)
);

CREATE INDEX idx_classes_academic_year ON classes (academic_year_id);
CREATE INDEX idx_classes_programme ON classes (programme_id);

CREATE TRIGGER trg_classes_set_updated_at
    BEFORE UPDATE ON classes
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE class_subjects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id    UUID NOT NULL REFERENCES classes (id) ON DELETE CASCADE,
    subject_id  UUID NOT NULL REFERENCES subjects (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (class_id, subject_id)
);

CREATE INDEX idx_class_subjects_subject_id ON class_subjects (subject_id);
