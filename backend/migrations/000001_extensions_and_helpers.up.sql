-- Enables gen_random_uuid(), used as the default for every primary key in
-- the schema (see section 7: "Use UUID primary keys").
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Generic trigger function used by every table that has an `updated_at`
-- column. Rather than re-declaring this logic per table, each later
-- migration creates a trigger that calls this function:
--
--   CREATE TRIGGER trg_<table>_set_updated_at
--       BEFORE UPDATE ON <table>
--       FOR EACH ROW
--       EXECUTE FUNCTION set_updated_at();
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
