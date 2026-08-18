DROP TABLE IF EXISTS `biz_deploy_host_lease`;

ALTER TABLE `biz_deploy_task_host`
  DROP COLUMN `resolved_at`,
  DROP COLUMN `report_key`,
  DROP COLUMN `business_scope_id`,
  DROP COLUMN `ssh_port`;

ALTER TABLE `biz_deploy_task`
  DROP COLUMN `start_request_key`,
  DROP COLUMN `target_snapshot`,
  DROP COLUMN `execution_snapshot`;
