-- Roll back the inviter-tracking column (see 000016 for rationale).

DROP INDEX IF EXISTS idx_users_invited_by;
ALTER TABLE users
    DROP COLUMN IF EXISTS invited_by;
