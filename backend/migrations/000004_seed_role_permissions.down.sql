-- Clears ALL role_permissions rows, including any assignments made via the
-- admin API (POST/DELETE /api/v1/roles/:id/permissions) after this seed
-- ran, not just the ones this migration added. Rolling back a seed
-- migration is a blunt instrument for that reason — if the mapping has
-- been customized since seeding, back it up (SELECT * FROM role_permissions)
-- before running this down migration.
DELETE FROM role_permissions;
