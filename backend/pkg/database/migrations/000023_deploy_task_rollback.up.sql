-- Add rollback support columns to biz_deploy_task.
-- These fields were introduced by the deploy rollback feature (RollbackTask,
-- TriggerAutoRollback) but the model-only change was never migrated. Adding
-- them here keeps the schema in sync with the GORM model.
ALTER TABLE `biz_deploy_task`
  ADD COLUMN `rollback_enabled` TINYINT(1) NOT NULL DEFAULT 0,
  ADD COLUMN `rollback_task_id` BIGINT UNSIGNED NULL,
  ADD COLUMN `source_task_id` BIGINT UNSIGNED NULL,
  ADD COLUMN `rollback_strategy` VARCHAR(32) NOT NULL DEFAULT '',
  ADD COLUMN `rollback_snapshot` JSON NULL;

CREATE INDEX `idx_deploy_task_rollback_task` ON `biz_deploy_task` (`rollback_task_id`);
CREATE INDEX `idx_deploy_task_source_task` ON `biz_deploy_task` (`source_task_id`);
