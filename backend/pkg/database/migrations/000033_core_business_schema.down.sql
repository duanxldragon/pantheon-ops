-- Non-destructive baseline rollback: retain business tables and data.
-- The up migration may have encountered pre-existing tables, so dropping them
-- cannot distinguish newly installed schema from a consumer database upgrade.
-- Restore a verified backup for a destructive schema downgrade.
SELECT 1;
