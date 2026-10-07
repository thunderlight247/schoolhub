-- Assessments and results (milestone M12).
--
-- Two tables, same layering as attendance (M11):
--   * assessments: one graded event — which class, which subject, which
--     term, what kind of assessment, when it happened, and the maximum
--     achievable score. The assessment is the header a teacher sets up
--     once; results hang off it.
--   * results: one row per student per assessment — the score itself, an
--     optional letter grade, and optional remarks.
--
-- Deliberate choices:
--   * class_id, subject_id, term_id are ON DELETE RESTRICT: results are
--     the official institutional record, exactly like attendance (M11) and
--     enrollments (M9). The service layer checks that the subject is in the
--     class curriculum (class_subjects, M8), the term belongs to the
--     class's academic year, and the assessment_date falls inside the term —
--     the same chain validation as attendance, without duplicating it in
--     constraints that can't span tables.
--   * score is NUMERIC, not a float column: marks like 87.5 are exact
--     decimal values. score may be NULL — a result row can exist before it
--     is marked. There is deliberately no CHECK tying score to the
--     assessment's max_score: PostgreSQL CHECKs cannot reference another
--     table, and the service enforces 0 <= score <= max_score on entry and
--     correction instead.
--   * grade is free text, not a computed column: grading scales vary by
--     institution, and no scale was specified. Computing letter grades from
--     percentages is deferred until a scale is configured (a later
--     milestone can backfill it).
--   * UNIQUE (class_id, subject_id, term_id, title): one "Quiz 1" per
--     class-subject-term. Distinct terms can reuse titles freely.
--   * assessment_id ON DELETE CASCADE on results: a result is meaningless
--     without its assessment. student_id ON DELETE CASCADE matches
--     enrollments (M9) and attendance (M11) — students are archived, not
--     deleted.
--   * No UNIQUE on (assessment_id, student_id) and score: an unmarked row
--     (NULL score) and a marked row must not coexist, hence the plain
--     unique on the pair, with the service forbidding re-entry.

CREATE TABLE assessments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id        UUID NOT NULL REFERENCES classes (id) ON DELETE RESTRICT,
    subject_id      UUID NOT NULL REFERENCES subjects (id) ON DELETE RESTRICT,
    term_id         UUID NOT NULL REFERENCES terms (id) ON DELETE RESTRICT,
    title           TEXT NOT NULL,
    type            TEXT NOT NULL DEFAULT 'TEST' CHECK (type IN ('QUIZ', 'TEST', 'ASSIGNMENT', 'PROJECT', 'MIDTERM', 'FINAL')),
    max_score       NUMERIC(6,2) NOT NULL CHECK (max_score > 0),
    weight          NUMERIC(5,2) CHECK (weight IS NULL OR (weight >= 0 AND weight <= 100)),
    assessment_date DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (class_id, subject_id, term_id, title)
);

CREATE INDEX idx_assessments_class_id  ON assessments (class_id);
CREATE INDEX idx_assessments_subject_id ON assessments (subject_id);
CREATE INDEX idx_assessments_term_id   ON assessments (term_id);
CREATE INDEX idx_assessments_date      ON assessments (assessment_date);

CREATE TRIGGER trg_assessments_set_updated_at
    BEFORE UPDATE ON assessments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE results (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    student_id    UUID NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    score         NUMERIC(6,2),
    grade         TEXT,
    remarks       TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (assessment_id, student_id)
);

CREATE INDEX idx_results_assessment_id ON results (assessment_id);
CREATE INDEX idx_results_student_id    ON results (student_id);

CREATE TRIGGER trg_results_set_updated_at
    BEFORE UPDATE ON results
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
