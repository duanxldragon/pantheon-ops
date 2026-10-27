SET @has_index := (SELECT COUNT(*) FROM information_schema.statistics
                   WHERE table_schema = DATABASE()
                     AND table_name = 'permission_workbench_remediation_event'
                     AND index_name = 'idx_permission_remediation_tenant_created');
SET @s := IF(@has_index > 0,
  'ALTER TABLE `permission_workbench_remediation_event` DROP INDEX `idx_permission_remediation_tenant_created`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @has_tenant_id := (SELECT COUNT(*) FROM information_schema.columns
                       WHERE table_schema = DATABASE()
                         AND table_name = 'permission_workbench_remediation_event'
                         AND column_name = 'tenant_id');
SET @s := IF(@has_tenant_id > 0,
  'ALTER TABLE `permission_workbench_remediation_event` DROP COLUMN `tenant_id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
