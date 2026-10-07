-- Announcements (milestone M13).
--
-- Announcements provide institutional communication channels:
--   * Broadcasts: school-wide messages visible to all users (target_type = 'ALL')
--   * Role-targeted: messages scoped to specific roles, e.g. TEACHER, PARENT, STUDENT, STAFF (target_type = 'ROLE')
--   * Class-targeted: messages scoped to a specific class cohort (target_type = 'CLASS')
--
-- Draft/Publish status flow:
--   * DRAFT: initial work in progress, only visible to the author and admins
--   * PUBLISHED: publicly broadcast to the resolved audience
--   * ARCHIVED: no longer active/displayed in active feeds
--
-- Audience resolution:
--   * Teachers see announcements for ALL, ROLE=TEACHER, and classes they teach
--   * Students see announcements for ALL, ROLE=STUDENT, and classes they are actively enrolled in
--   * Parents see announcements for ALL, ROLE=PARENT, and classes their children are enrolled in
--   * Admins can view and manage all announcements

CREATE TABLE announcements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT NOT NULL,
    content         TEXT NOT NULL,
    author_id       UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED')),
    target_type     TEXT NOT NULL DEFAULT 'ALL' CHECK (target_type IN ('ALL', 'ROLE', 'CLASS')),
    target_role     TEXT CHECK (target_role IS NULL OR target_role IN ('ADMIN', 'TEACHER', 'STAFF', 'PARENT', 'STUDENT')),
    target_class_id UUID REFERENCES classes (id) ON DELETE CASCADE,
    priority        TEXT NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW', 'NORMAL', 'HIGH', 'URGENT')),
    published_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_announcements_target_integrity CHECK (
        (target_type = 'ALL' AND target_role IS NULL AND target_class_id IS NULL) OR
        (target_type = 'ROLE' AND target_role IS NOT NULL AND target_class_id IS NULL) OR
        (target_type = 'CLASS' AND target_class_id IS NOT NULL AND target_role IS NULL)
    )
);

CREATE INDEX idx_announcements_author_id       ON announcements (author_id);
CREATE INDEX idx_announcements_status          ON announcements (status);
CREATE INDEX idx_announcements_target_type     ON announcements (target_type);
CREATE INDEX idx_announcements_target_role     ON announcements (target_role);
CREATE INDEX idx_announcements_target_class_id ON announcements (target_class_id);
CREATE INDEX idx_announcements_created_at      ON announcements (created_at DESC);
CREATE INDEX idx_announcements_published_at    ON announcements (published_at DESC);

CREATE TRIGGER trg_announcements_set_updated_at
    BEFORE UPDATE ON announcements
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
