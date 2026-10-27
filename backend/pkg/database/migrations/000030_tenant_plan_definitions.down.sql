-- Rollback of 000030_tenant_plan_definitions.up.sql: drop the plan table.
--
-- `tenants.plan` is deliberately left alone. The column predates this table and
-- still carries the code each tenant was created with, so rolling back costs the
-- deployment its edits to thresholds — which is what a rollback of a
-- configuration table means — but no tenant loses its identity, and
-- GetDefaultQuota resumes as the only source once the table is gone.
DROP TABLE IF EXISTS `tenant_plans`;
