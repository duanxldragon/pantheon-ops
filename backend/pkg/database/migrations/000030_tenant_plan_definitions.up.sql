-- Tenant plan definitions (contract TENANT_CONTRACT_V1 §2.4)
--
-- WHY THIS TABLE EXISTS
-- ---------------------
-- The quota thresholds behind `tenants.plan` were a switch statement in
-- modules/system/iam/tenant/tenant_production.go (GetDefaultQuota). Nothing
-- could change them without a code release, so the platform console had no way
-- to state what a plan actually grants — which is the single number an operator
-- asks for when asked "how big is this tenant".
--
-- SEMANTICS
-- ---------
-- Platform-global table: no tenant_id column, exactly like `tenants` itself.
-- Tenant confinement in this module is applied by explicit WithTenantScope calls
-- in the services that need it, not by a global plugin, so an unscoped table is
-- the correct shape (pkg/tenant.WithTenantScope). Adding tenant_id here would
-- make the platform console's own reads unwinnable.
--
-- A limit of -1 means unlimited. 0 is rejected by the service on purpose: every
-- consumer distinguishes "no ceiling" from "ceiling reached", and 0 would let a
-- stored plan blur the two.
--
-- The four seeded rows carry the values the switch statement produced, so the
-- upgrade is a no-op for every existing tenant. The seed is guarded per code,
-- never an UPDATE: once the console can edit a threshold, a migration that
-- rewrote values on replay would silently undo the operator (replay is a real
-- state here — see the baseline note in 000024).
--
-- Removing a plan is not modelled. Retirement is `status = 'disabled'`, refused
-- while tenants still carry the code (PlanStore.Update -> ErrPlanInUse); a row
-- that vanished under a tenant would leave that tenant's quota undefined.

CREATE TABLE IF NOT EXISTS `tenant_plans` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(32) NOT NULL,
  `name` VARCHAR(64) NOT NULL DEFAULT '',
  `max_users` INT NOT NULL DEFAULT 0,
  `max_roles` INT NOT NULL DEFAULT 0,
  `max_depts` INT NOT NULL DEFAULT 0,
  `max_api_keys` INT NOT NULL DEFAULT 0,
  `max_storage_mb` INT NOT NULL DEFAULT 0,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `sort` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_tenant_plan_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `tenant_plans`
  (`code`, `name`, `max_users`, `max_roles`, `max_depts`, `max_api_keys`, `max_storage_mb`, `status`, `sort`, `created_at`, `updated_at`)
SELECT 'free', 'Free', 5, 3, 5, 2, 100, 'active', 10, NOW(3), NOW(3)
WHERE NOT EXISTS (SELECT 1 FROM `tenant_plans` WHERE `code` = 'free');

INSERT INTO `tenant_plans`
  (`code`, `name`, `max_users`, `max_roles`, `max_depts`, `max_api_keys`, `max_storage_mb`, `status`, `sort`, `created_at`, `updated_at`)
SELECT 'basic', 'Basic', 20, 10, 20, 10, 1000, 'active', 20, NOW(3), NOW(3)
WHERE NOT EXISTS (SELECT 1 FROM `tenant_plans` WHERE `code` = 'basic');

INSERT INTO `tenant_plans`
  (`code`, `name`, `max_users`, `max_roles`, `max_depts`, `max_api_keys`, `max_storage_mb`, `status`, `sort`, `created_at`, `updated_at`)
SELECT 'professional', 'Professional', 100, 50, 100, 50, 10000, 'active', 30, NOW(3), NOW(3)
WHERE NOT EXISTS (SELECT 1 FROM `tenant_plans` WHERE `code` = 'professional');

INSERT INTO `tenant_plans`
  (`code`, `name`, `max_users`, `max_roles`, `max_depts`, `max_api_keys`, `max_storage_mb`, `status`, `sort`, `created_at`, `updated_at`)
SELECT 'enterprise', 'Enterprise', -1, -1, -1, -1, -1, 'active', 40, NOW(3), NOW(3)
WHERE NOT EXISTS (SELECT 1 FROM `tenant_plans` WHERE `code` = 'enterprise');
