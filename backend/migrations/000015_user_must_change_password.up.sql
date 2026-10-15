-- M34: forced first-login password change.
--
-- The bootstrap master admin is provisioned with a temporary password the
-- developer knows; the flag below forces a change at first login and the
-- auth middleware rejects everything except change/logout/me until the
-- flag is cleared (so the bootstrap password cannot be reused by anyone,
-- developer included, after the first session).
ALTER TABLE users
    ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT FALSE;
