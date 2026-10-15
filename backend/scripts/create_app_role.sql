-- Least-privilege database access (section 28, milestone M20).
--
-- Every migration (and sqlc generate) runs as the `schoolhub` role, which
-- OWNS every table it created — full DDL rights (CREATE/DROP/ALTER) on its
-- own objects. The running API has been using that SAME role via
-- DATABASE_URL since M0. sqlc's parameterized queries make SQL injection
-- unlikely, but least-privilege is a defense-in-depth layer, not a bet
-- that the first layer never fails: a bug or a compromised process
-- shouldn't be able to DROP TABLE just because the app happens to run as
-- the owning role.
--
-- This script creates `schoolhub_app`, a role with only SELECT/INSERT/
-- UPDATE/DELETE — no CREATE/DROP/ALTER on anything. docker-compose.yml's
-- `backend` service now connects as this role; `migrate` and `sqlc`
-- continue to run as the owning `schoolhub` role, since schema changes
-- are exactly what they're for.
--
-- NOT a golang-migrate migration: migrate has no mechanism to pass a
-- password in from an environment variable, and role/grant management is
-- cluster-level administration, not application schema. Run once per
-- database, via psql's -v flag (see README's M20 section for the full
-- command, including how to apply this to a database that already existed
-- before this script).

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'schoolhub_app') THEN
        CREATE ROLE schoolhub_app LOGIN PASSWORD :'app_password';
    ELSE
        ALTER ROLE schoolhub_app LOGIN PASSWORD :'app_password';
    END IF;
END
$$;

GRANT CONNECT ON DATABASE schoolhub TO schoolhub_app;
GRANT USAGE ON SCHEMA public TO schoolhub_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO schoolhub_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO schoolhub_app;

-- Ensures tables/sequences created by FUTURE migrations (M21 onward,
-- still run as the owning `schoolhub` role) automatically grant
-- schoolhub_app the same DML rights, without every future migration
-- needing its own GRANT statement.
ALTER DEFAULT PRIVILEGES FOR ROLE schoolhub IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO schoolhub_app;
ALTER DEFAULT PRIVILEGES FOR ROLE schoolhub IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO schoolhub_app;
