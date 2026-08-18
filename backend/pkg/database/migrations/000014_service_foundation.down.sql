ALTER TABLE `biz_k8s_release`
  DROP KEY `idx_k8s_release_service_instance`,
  DROP COLUMN `service_instance_name`,
  DROP COLUMN `service_name`,
  DROP COLUMN `service_instance_id`,
  DROP COLUMN `service_id`;

ALTER TABLE `biz_deploy_task`
  DROP KEY `idx_deploy_task_service_instance`,
  DROP COLUMN `service_instance_name`,
  DROP COLUMN `service_name`,
  DROP COLUMN `service_instance_id`,
  DROP COLUMN `service_id`;

DROP TABLE IF EXISTS `biz_service_instance`;
DROP TABLE IF EXISTS `biz_service`;
DROP TABLE IF EXISTS `biz_application`;
