-- Pantheon Base: Initial Schema Migration
-- This migration captures the full baseline schema that was previously managed by GORM AutoMigrate.
-- All tables use InnoDB engine with utf8mb4 charset.

-- ============================================================================
-- Auth Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_user_session` (
  `session_id` VARCHAR(64) NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `refresh_jti` VARCHAR(64) NOT NULL,
  `user_agent` VARCHAR(512) DEFAULT '',
  `ip` VARCHAR(64) DEFAULT '',
  `expires_at` DATETIME(3) NOT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`session_id`),
  INDEX `idx_system_user_session_user_id` (`user_id`),
  INDEX `idx_system_user_session_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_log_login` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `request_id` VARCHAR(64) DEFAULT '',
  `username` VARCHAR(64) DEFAULT '',
  `ip` VARCHAR(64) DEFAULT '',
  `location` VARCHAR(128) DEFAULT '',
  `browser` VARCHAR(128) DEFAULT '',
  `os` VARCHAR(128) DEFAULT '',
  `status` TINYINT DEFAULT 0,
  `msg` VARCHAR(255) DEFAULT '',
  `login_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_system_log_login_request_id` (`request_id`),
  INDEX `idx_system_log_login_username` (`username`),
  INDEX `idx_system_log_login_login_at` (`login_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_login_throttle` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `source_key` VARCHAR(191) NOT NULL,
  `failure_count` INT DEFAULT 0,
  `last_failure_at` DATETIME(3) DEFAULT NULL,
  `locked_until` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_login_throttle_source_key` (`source_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_auth_factor` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `factor_type` VARCHAR(32) NOT NULL DEFAULT 'totp',
  `secret` VARCHAR(255) NOT NULL,
  `verified` TINYINT DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_auth_factor_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_auth_mfa_challenge` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `challenge_id` VARCHAR(64) NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `factor_type` VARCHAR(32) NOT NULL DEFAULT 'totp',
  `verified` TINYINT DEFAULT 0,
  `expires_at` DATETIME(3) NOT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_auth_mfa_challenge_challenge_id` (`challenge_id`),
  INDEX `idx_system_auth_mfa_challenge_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_auth_security_event` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED DEFAULT NULL,
  `username` VARCHAR(64) DEFAULT '',
  `event_type` VARCHAR(64) NOT NULL,
  `ip` VARCHAR(64) DEFAULT '',
  `user_agent` VARCHAR(512) DEFAULT '',
  `detail` TEXT,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_system_auth_security_event_user_id` (`user_id`),
  INDEX `idx_system_auth_security_event_username` (`username`),
  INDEX `idx_system_auth_security_event_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_user_password_history` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `password_hash` VARCHAR(255) NOT NULL,
  `changed_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_user_password_history_user_changed` (`user_id`, `changed_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- User Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_user` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(64) NOT NULL,
  `password` VARCHAR(255) NOT NULL,
  `nickname` VARCHAR(64) DEFAULT '',
  `avatar` VARCHAR(512) DEFAULT '',
  `email` VARCHAR(128) DEFAULT '',
  `phone` VARCHAR(32) DEFAULT '',
  `dept_id` BIGINT UNSIGNED DEFAULT 0,
  `post_id` BIGINT UNSIGNED DEFAULT 0,
  `status` TINYINT DEFAULT 1,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_user_username` (`username`),
  INDEX `idx_system_user_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_user_role` (
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (`user_id`, `role_id`),
  INDEX `idx_system_user_role_user` (`user_id`),
  INDEX `idx_system_user_role_role` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_user_profile_ext` (
  `user_id` BIGINT UNSIGNED NOT NULL,
  `profile_json` TEXT,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Role Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_role` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_name` VARCHAR(64) NOT NULL,
  `role_key` VARCHAR(64) NOT NULL,
  `sort` INT DEFAULT 0,
  `data_scope` VARCHAR(32) DEFAULT 'all',
  `status` TINYINT DEFAULT 1,
  `remark` VARCHAR(255) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_role_role_key` (`role_key`),
  INDEX `idx_system_role_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_role_permission` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `permission_key` VARCHAR(128) NOT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_role_permission_unique` (`role_id`, `permission_key`),
  INDEX `idx_system_role_permission_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_role_menu` (
  `role_id` BIGINT UNSIGNED NOT NULL,
  `menu_id` BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (`role_id`, `menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Menu Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_menu` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `parent_id` BIGINT UNSIGNED DEFAULT 0,
  `title_key` VARCHAR(64) NOT NULL,
  `icon` VARCHAR(64) DEFAULT '',
  `path` VARCHAR(255) DEFAULT '',
  `component` VARCHAR(255) DEFAULT '',
  `redirect` VARCHAR(255) DEFAULT '',
  `menu_type` TINYINT DEFAULT 0,
  `permission` VARCHAR(128) DEFAULT '',
  `sort` INT DEFAULT 0,
  `status` TINYINT DEFAULT 1,
  `visible` TINYINT DEFAULT 1,
  `hide_in_nav` INT DEFAULT 0,
  `cache` TINYINT DEFAULT 0,
  `remark` VARCHAR(255) DEFAULT '',
  `module` VARCHAR(64) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_system_menu_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Organization Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_dept` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `parent_id` BIGINT UNSIGNED DEFAULT 0,
  `ancestors` VARCHAR(255) DEFAULT '',
  `dept_name` VARCHAR(64) NOT NULL,
  `dept_code` VARCHAR(64) DEFAULT '',
  `leader_user_id` BIGINT UNSIGNED DEFAULT 0,
  `sort` INT DEFAULT 0,
  `status` TINYINT DEFAULT 1,
  `remark` VARCHAR(255) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_dept_dept_code` (`dept_code`),
  INDEX `idx_system_dept_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_post` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `dept_id` BIGINT UNSIGNED DEFAULT 0,
  `post_code` VARCHAR(64) NOT NULL,
  `post_name` VARCHAR(64) NOT NULL,
  `sort` INT DEFAULT 0,
  `status` TINYINT DEFAULT 1,
  `remark` VARCHAR(255) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_post_post_code` (`post_code`),
  INDEX `idx_system_post_dept_id` (`dept_id`),
  INDEX `idx_system_post_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Setting Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_setting` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `setting_key` VARCHAR(128) NOT NULL,
  `setting_value` TEXT,
  `group_key` VARCHAR(64) DEFAULT 'general',
  `label_key` VARCHAR(128) DEFAULT '',
  `description` VARCHAR(255) DEFAULT '',
  `value_type` VARCHAR(32) DEFAULT 'string',
  `sort` INT DEFAULT 0,
  `is_public` TINYINT DEFAULT 0,
  `is_readonly` TINYINT DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_setting_setting_key` (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_setting_audit_log` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `setting_key` VARCHAR(128) DEFAULT '',
  `action` VARCHAR(32) DEFAULT '',
  `old_value` TEXT,
  `new_value` TEXT,
  `oper_param` JSON,
  `operator_id` BIGINT UNSIGNED DEFAULT 0,
  `operator_name` VARCHAR(64) DEFAULT '',
  `operator_ip` VARCHAR(64) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_system_setting_audit_log_key` (`setting_key`),
  INDEX `idx_system_setting_audit_log_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Audit Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_log_oper` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `request_id` VARCHAR(64) DEFAULT '',
  `title` VARCHAR(64) DEFAULT '',
  `method` VARCHAR(16) DEFAULT '',
  `path` VARCHAR(255) DEFAULT '',
  `action` VARCHAR(128) DEFAULT '',
  `status` INT DEFAULT 0,
  `oper_param` TEXT,
  `result` TEXT,
  `operator_id` BIGINT UNSIGNED DEFAULT 0,
  `operator_name` VARCHAR(64) DEFAULT '',
  `operator_ip` VARCHAR(64) DEFAULT '',
  `user_agent` VARCHAR(512) DEFAULT '',
  `cost_ms` INT DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_system_log_oper_request_id` (`request_id`),
  INDEX `idx_system_log_oper_operator_id` (`operator_id`),
  INDEX `idx_system_log_oper_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- I18n Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_i18n` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `module` VARCHAR(64) NOT NULL,
  `group_name` VARCHAR(64) NOT NULL DEFAULT 'messages',
  `locale_key` VARCHAR(128) NOT NULL,
  `locale` VARCHAR(16) NOT NULL DEFAULT 'zh-CN',
  `value` TEXT,
  `is_builtin` TINYINT DEFAULT 0,
  `status` TINYINT DEFAULT 1,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_system_i18n_module_key` (`module`, `locale_key`),
  INDEX `idx_system_i18n_module_group` (`module`, `group_name`),
  INDEX `idx_system_i18n_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Dict Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_dict_type` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `dict_code` VARCHAR(64) NOT NULL,
  `dict_name` VARCHAR(64) NOT NULL,
  `status` TINYINT DEFAULT 1,
  `remark` VARCHAR(255) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_dict_type_dict_code` (`dict_code`),
  INDEX `idx_system_dict_type_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_dict_item` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `dict_code` VARCHAR(64) NOT NULL,
  `item_label_key` VARCHAR(128) NOT NULL,
  `item_value` VARCHAR(255) NOT NULL,
  `sort` INT DEFAULT 0,
  `status` TINYINT DEFAULT 1,
  `remark` VARCHAR(255) DEFAULT '',
  `tag_type` VARCHAR(32) DEFAULT '',
  `css_class` VARCHAR(64) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_dict_item_code_sort` (`dict_code`, `sort`),
  INDEX `idx_system_dict_item_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Permission Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_role_data_scope` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_key` VARCHAR(64) NOT NULL,
  `mode` VARCHAR(32) NOT NULL DEFAULT 'all',
  `scope_defs` TEXT,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_role_data_scope_role_key` (`role_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `permission_workbench_remediation_event` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_key` VARCHAR(64) NOT NULL,
  `issue_type` VARCHAR(32) NOT NULL,
  `severity` VARCHAR(16) NOT NULL DEFAULT 'medium',
  `detail` TEXT,
  `remediated` TINYINT DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_permission_remediation_role_created` (`role_key`, `created_at`),
  INDEX `idx_permission_remediation_issue_type` (`issue_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `permission_role_data_scope_policy` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_key` VARCHAR(64) NOT NULL,
  `mode` VARCHAR(32) NOT NULL DEFAULT 'all',
  `scope_defs` TEXT,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_permission_role_data_scope_policy_role_key` (`role_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Casbin Rule Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS `casbin_rule` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `ptype` VARCHAR(100) DEFAULT '',
  `v0` VARCHAR(100) DEFAULT '',
  `v1` VARCHAR(100) DEFAULT '',
  `v2` VARCHAR(100) DEFAULT '',
  `v3` VARCHAR(100) DEFAULT '',
  `v4` VARCHAR(100) DEFAULT '',
  `v5` VARCHAR(100) DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Refresh Sync Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS `system_refresh_version` (
  `topic` VARCHAR(64) NOT NULL,
  `version` BIGINT NOT NULL DEFAULT 0,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`topic`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Dynamic Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `module_registration` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) DEFAULT '',
  `display_name` VARCHAR(128) DEFAULT '',
  `module_type` VARCHAR(32) DEFAULT '',
  `status` TINYINT DEFAULT 1,
  `registered_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_module_registration_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Generator Module Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS `generator_datasource` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `driver` VARCHAR(32) NOT NULL DEFAULT 'mysql',
  `dsn` VARCHAR(512) DEFAULT '',
  `status` TINYINT DEFAULT 1,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_generator_datasource_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
