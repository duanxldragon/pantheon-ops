-- Pantheon Base: Initial Schema Rollback
-- Drop all tables in reverse dependency order.

DROP TABLE IF EXISTS `generator_datasource`;
DROP TABLE IF EXISTS `module_registration`;
DROP TABLE IF EXISTS `system_refresh_version`;
DROP TABLE IF EXISTS `casbin_rule`;
DROP TABLE IF EXISTS `permission_role_data_scope_policy`;
DROP TABLE IF EXISTS `permission_workbench_remediation_event`;
DROP TABLE IF EXISTS `system_role_data_scope`;
DROP TABLE IF EXISTS `system_dict_item`;
DROP TABLE IF EXISTS `system_dict_type`;
DROP TABLE IF EXISTS `system_i18n`;
DROP TABLE IF EXISTS `system_log_oper`;
DROP TABLE IF EXISTS `system_setting_audit_log`;
DROP TABLE IF EXISTS `system_setting`;
DROP TABLE IF EXISTS `system_post`;
DROP TABLE IF EXISTS `system_dept`;
DROP TABLE IF EXISTS `system_menu`;
DROP TABLE IF EXISTS `system_role_menu`;
DROP TABLE IF EXISTS `system_role_permission`;
DROP TABLE IF EXISTS `system_role`;
DROP TABLE IF EXISTS `system_user_profile_ext`;
DROP TABLE IF EXISTS `system_user_role`;
DROP TABLE IF EXISTS `system_user`;
DROP TABLE IF EXISTS `system_user_password_history`;
DROP TABLE IF EXISTS `system_auth_security_event`;
DROP TABLE IF EXISTS `system_auth_mfa_challenge`;
DROP TABLE IF EXISTS `system_auth_factor`;
DROP TABLE IF EXISTS `system_login_throttle`;
DROP TABLE IF EXISTS `system_log_login`;
DROP TABLE IF EXISTS `system_user_session`;
