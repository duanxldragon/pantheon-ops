ALTER TABLE `system_generator_datasource`
  DROP COLUMN `last_check_error`,
  DROP COLUMN `last_check_status`,
  DROP COLUMN `last_checked_at`,
  DROP COLUMN `remark`,
  DROP COLUMN `readonly_scope`,
  DROP COLUMN `password_encrypted`,
  DROP COLUMN `username`,
  DROP COLUMN `database_name`,
  DROP COLUMN `port`,
  DROP COLUMN `host`;

ALTER TABLE `system_dept`
  ADD COLUMN `dept_code` VARCHAR(64) DEFAULT '' AFTER `dept_name`,
  ADD UNIQUE INDEX `idx_system_dept_dept_code` (`dept_code`);
