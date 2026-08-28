-- Drop the changed checkpoint column from deploy task steps.
ALTER TABLE `biz_deploy_task_step`
  DROP COLUMN `changed`;
