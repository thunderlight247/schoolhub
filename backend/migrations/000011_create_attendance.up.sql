-- Attendance (milestone M11).
--
-- Attendance is modeled in two levels:
--   * attendance_sessions: one register-taking event — which class, which
--     subject, which term, on which date. The session is what a teacher
--     captures in one sitting ("take the register"), and what summaries
--     aggregate over.
--   * attendance_records: one row per student per session, holding the
--     mark itself (PRESENT / ABSENT / LATE / EXCUSED) and an optional note.
--
-- Deliberate choices:
--   * A session table instead of flat per-student rows: capture is naturally
--     whole-roster (the teacher marks every student at once), the UNIQUE
--     constraints stay simple (one register per class/subject/term/day;
--     one record per student per session), and later reporting ("all
--     sessions for this class in term X") reads directly off the session.
--   * class_id, subject_id, term_id are ON DELETE RESTRICT: attendance is
--     the official institutional record, not derived coverage data like
--     teacher_assignments (M10) — deleting a class that has attendance
--     history must be blocked, exactly like enrollments (M9) restrict the
--     same FKs. The class -> academic_year link (000008) plus the term's
--     academic_year_id give the temporal scoping; the service layer checks
--     they agree rather than storing a fourth FK.
--   * session_id ON DELETE CASCADE on records: a record is meaningless
--     without its session. student_id ON DELETE CASCADE matches the
--     enrollments table (M9) — students are archived, not deleted, so this
--     only fires on direct row removal.
--   * No recorded_by column in M11: the actor is already written to the
--     audit log (ATTENDANCE_RECORDED) inside the capture transaction; a
--     denormalized "who took the register" column is deferred until the
--     teacher portal (M15) actually needs to render it.

CREATE TABLE attendance_sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id     UUID NOT NULL REFERENCES classes (id) ON DELETE RESTRICT,
    subject_id   UUID NOT NULL REFERENCES subjects (id) ON DELETE RESTRICT,
    term_id      UUID NOT NULL REFERENCES terms (id) ON DELETE RESTRICT,
    session_date DATE NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One register per class-subject-term-day: retaking attendance for the
    -- same teaching slot updates the existing records (via PATCH on the
    -- record rows), it never silently opens a second register.
    UNIQUE (class_id, subject_id, term_id, session_date)
);

CREATE INDEX idx_attendance_sessions_class_id  ON attendance_sessions (class_id);
CREATE INDEX idx_attendance_sessions_subject_id ON attendance_sessions (subject_id);
CREATE INDEX idx_attendance_sessions_term_id   ON attendance_sessions (term_id);
CREATE INDEX idx_attendance_sessions_date      ON attendance_sessions (session_date);

CREATE TRIGGER trg_attendance_sessions_set_updated_at
    BEFORE UPDATE ON attendance_sessions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE attendance_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES attendance_sessions (id) ON DELETE CASCADE,
    student_id  UUID NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    status      TEXT NOT NULL CHECK (status IN ('PRESENT', 'ABSENT', 'LATE', 'EXCUSED')),
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (session_id, student_id)
);

CREATE INDEX idx_attendance_records_session_id ON attendance_records (session_id);
CREATE INDEX idx_attendance_records_student_id ON attendance_records (student_id);
CREATE INDEX idx_attendance_records_status     ON attendance_records (status);

CREATE TRIGGER trg_attendance_records_set_updated_at
    BEFORE UPDATE ON attendance_records
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
