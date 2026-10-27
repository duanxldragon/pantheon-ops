-- Rollback for 000012_business_schema. Business tables are removed in reverse
-- ownership/reference order. This down migration is intended for controlled
-- rollback before production data is accepted; the up migration is additive.
DROP TABLE IF EXISTS `biz_k8s_release`;
DROP TABLE IF EXISTS `biz_k8s_cluster`;
DROP TABLE IF EXISTS `biz_deploy_task_host`;
DROP TABLE IF EXISTS `biz_deploy_task`;
DROP TABLE IF EXISTS `biz_deploy_template_step`;
DROP TABLE IF EXISTS `biz_deploy_template`;
DROP TABLE IF EXISTS `biz_deploy_package`;
DROP TABLE IF EXISTS `biz_cmdb_label_schema`;
DROP TABLE IF EXISTS `biz_cmdb_group`;
DROP TABLE IF EXISTS `biz_cmdb_host`;
DROP TABLE IF EXISTS `biz_business_scope`;
