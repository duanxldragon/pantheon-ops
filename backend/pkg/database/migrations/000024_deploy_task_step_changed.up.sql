-- Add the changed checkpoint column to deploy task steps.
-- DeployTaskStep.Changed (gorm default:false) was added to the model for
-- checkpoint resume tracking but never migrated.
ALTER TABLE `biz_deploy_task_step`
  ADD COLUMN `changed` TINYINT(1) NOT NULL DEFAULT 0;
