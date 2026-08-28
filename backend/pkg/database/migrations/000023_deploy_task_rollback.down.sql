-- Drop rollback support columns from biz_deploy_task.
ALTER TABLE `biz_deploy_task`
  DROP INDEX `idx_deploy_task_rollback_task`,
  DROP INDEX `idx_deploy_task_source_task`,
  DROP COLUMN `rollback_snapshot`,
  DROP COLUMN `rollback_strategy`,
  DROP COLUMN `source_task_id`,
  DROP COLUMN `rollback_task_id`,
  DROP COLUMN `rollback_enabled`;
