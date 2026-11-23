-- Retained down: tenant_id columns and tenant-first unique keys are the
-- multi-tenant ownership contract; dropping them would destroy tenant rows and
-- reintroduce cross-tenant name collisions. Roll back by restoring a pre-T04
-- backup in an isolated environment.
SELECT 1;
