-- Align the original versioned migration baseline with the current runtime schema
-- used by the smoke suite and backend startup.

ALTER TABLE `system_menu`
  ADD COLUMN `page_perm` VARCHAR(128) DEFAULT '' AFTER `component`,
  ADD COLUMN `perms` VARCHAR(128) DEFAULT '' AFTER `page_perm`,
  ADD COLUMN `type` CHAR(1) DEFAULT 'M' AFTER `perms`,
  ADD COLUMN `route_name` VARCHAR(128) DEFAULT '' AFTER `icon`,
  ADD COLUMN `is_visible` TINYINT DEFAULT 1 AFTER `sort`,
  ADD COLUMN `is_cache` TINYINT DEFAULT 0 AFTER `is_visible`,
  ADD COLUMN `is_external` TINYINT DEFAULT 0 AFTER `is_cache`,
  ADD COLUMN `active_menu` VARCHAR(255) DEFAULT '' AFTER `is_external`;

UPDATE `system_menu`
SET
  `page_perm` = CASE
    WHEN COALESCE(`page_perm`, '') = '' AND COALESCE(`permission`, '') <> '' THEN `permission`
    ELSE COALESCE(`page_perm`, '')
  END,
  `type` = CASE
    WHEN COALESCE(`type`, '') <> '' THEN `type`
    WHEN COALESCE(`component`, '') <> '' THEN 'C'
    WHEN COALESCE(`permission`, '') <> '' THEN 'F'
    ELSE 'M'
  END,
  `is_visible` = CASE
    WHEN COALESCE(`is_visible`, 0) IN (0, 1) THEN `is_visible`
    ELSE COALESCE(`visible`, 1)
  END,
  `is_cache` = CASE
    WHEN COALESCE(`is_cache`, 0) IN (0, 1) THEN `is_cache`
    ELSE COALESCE(`cache`, 0)
  END;

ALTER TABLE `system_user`
  ADD COLUMN `preference_json` TEXT AFTER `phone`,
  ADD COLUMN `failed_login_attempts` INT DEFAULT 0 AFTER `status`,
  ADD COLUMN `login_locked_until` DATETIME(3) NULL AFTER `failed_login_attempts`;

ALTER TABLE `system_dept`
  ADD COLUMN `is_root` TINYINT DEFAULT 0 AFTER `ancestors`,
  ADD COLUMN `leader` VARCHAR(64) DEFAULT '' AFTER `leader_user_id`,
  ADD COLUMN `phone` VARCHAR(20) DEFAULT '' AFTER `leader`,
  ADD COLUMN `email` VARCHAR(128) DEFAULT '' AFTER `phone`;

ALTER TABLE `system_user_session`
  ADD COLUMN `refresh_expires_at` DATETIME(3) NULL AFTER `refresh_jti`,
  ADD COLUMN `last_refresh_at` DATETIME(3) NULL AFTER `refresh_expires_at`,
  ADD COLUMN `last_activity_at` DATETIME(3) NULL AFTER `last_refresh_at`,
  ADD COLUMN `last_ip` VARCHAR(128) DEFAULT '' AFTER `last_activity_at`,
  ADD COLUMN `revoked_at` DATETIME(3) NULL AFTER `user_agent`;

UPDATE `system_user_session`
SET
  `refresh_expires_at` = COALESCE(`refresh_expires_at`, `expires_at`),
  `last_ip` = CASE
    WHEN COALESCE(`last_ip`, '') = '' THEN COALESCE(`ip`, '')
    ELSE `last_ip`
  END;

ALTER TABLE `system_log_login`
  ADD COLUMN `ipaddr` VARCHAR(128) DEFAULT '' AFTER `username`,
  ADD COLUMN `login_location` VARCHAR(255) DEFAULT '' AFTER `ipaddr`,
  ADD COLUMN `login_time` DATETIME(3) NULL AFTER `msg`;

UPDATE `system_log_login`
SET
  `ipaddr` = CASE
    WHEN COALESCE(`ipaddr`, '') = '' THEN COALESCE(`ip`, '')
    ELSE `ipaddr`
  END,
  `login_location` = CASE
    WHEN COALESCE(`login_location`, '') = '' THEN COALESCE(`location`, '')
    ELSE `login_location`
  END,
  `login_time` = COALESCE(`login_time`, `login_at`);

ALTER TABLE `system_login_throttle`
  ADD COLUMN `window_started_at` DATETIME(3) NULL AFTER `failure_count`,
  ADD COLUMN `last_attempt_at` DATETIME(3) NULL AFTER `window_started_at`,
  ADD COLUMN `blocked_until` DATETIME(3) NULL AFTER `last_attempt_at`,
  ADD COLUMN `created_at` DATETIME(3) NULL AFTER `blocked_until`,
  ADD COLUMN `updated_at` DATETIME(3) NULL AFTER `created_at`;

UPDATE `system_login_throttle`
SET `blocked_until` = COALESCE(`blocked_until`, `locked_until`);

ALTER TABLE `system_auth_factor`
  ADD COLUMN `secret_encrypted` VARCHAR(512) DEFAULT '' AFTER `factor_type`,
  ADD COLUMN `enabled` TINYINT DEFAULT 1 AFTER `secret_encrypted`,
  ADD COLUMN `confirmed_at` DATETIME(3) NULL AFTER `enabled`;

UPDATE `system_auth_factor`
SET
  `secret_encrypted` = CASE
    WHEN COALESCE(`secret_encrypted`, '') = '' THEN COALESCE(`secret`, '')
    ELSE `secret_encrypted`
  END,
  `enabled` = CASE
    WHEN COALESCE(`enabled`, 0) IN (0, 1) THEN `enabled`
    ELSE COALESCE(`verified`, 0)
  END;

ALTER TABLE `system_auth_mfa_challenge`
  ADD COLUMN `purpose` VARCHAR(32) NOT NULL DEFAULT 'login' AFTER `user_id`,
  ADD COLUMN `secret_encrypted` VARCHAR(512) DEFAULT '' AFTER `purpose`,
  ADD COLUMN `setup_required` TINYINT DEFAULT 0 AFTER `secret_encrypted`,
  ADD COLUMN `consumed_at` DATETIME(3) NULL AFTER `expires_at`,
  ADD COLUMN `updated_at` DATETIME(3) NULL AFTER `created_at`;

ALTER TABLE `system_auth_security_event`
  ADD COLUMN `severity` VARCHAR(16) NOT NULL DEFAULT 'medium' AFTER `event_type`,
  ADD COLUMN `source_key` VARCHAR(191) DEFAULT '' AFTER `severity`,
  ADD COLUMN `message_key` VARCHAR(128) NOT NULL DEFAULT 'auth.securityEvent.unknown' AFTER `user_agent`,
  ADD COLUMN `metadata` TEXT AFTER `message_key`,
  ADD COLUMN `acknowledged_at` DATETIME(3) NULL AFTER `metadata`,
  ADD COLUMN `acknowledged_by` BIGINT UNSIGNED DEFAULT 0 AFTER `acknowledged_at`,
  ADD COLUMN `acknowledged_by_user` VARCHAR(64) DEFAULT '' AFTER `acknowledged_by`,
  ADD COLUMN `acknowledgement_note` VARCHAR(1000) DEFAULT '' AFTER `acknowledged_by_user`;

UPDATE `system_auth_security_event`
SET `metadata` = CASE
  WHEN COALESCE(`metadata`, '') = '' THEN COALESCE(`detail`, '')
  ELSE `metadata`
END;

ALTER TABLE `system_setting`
  ADD COLUMN `module` VARCHAR(64) NOT NULL DEFAULT 'system' AFTER `group_key`,
  ADD COLUMN `is_encrypted` TINYINT DEFAULT 0 AFTER `is_public`,
  ADD COLUMN `remark` VARCHAR(255) DEFAULT '' AFTER `is_encrypted`;

UPDATE `system_setting`
SET `remark` = CASE
  WHEN COALESCE(`remark`, '') <> '' THEN `remark`
  WHEN COALESCE(`label_key`, '') <> '' THEN `label_key`
  ELSE COALESCE(`description`, '')
END;

ALTER TABLE `system_i18n`
  ADD COLUMN `key` VARCHAR(128) DEFAULT '' AFTER `group_name`,
  ADD COLUMN `remark` VARCHAR(255) DEFAULT '' AFTER `value`,
  ADD COLUMN `lifecycle_status` VARCHAR(16) NOT NULL DEFAULT 'active' AFTER `remark`,
  ADD COLUMN `lifecycle_marked_at` DATETIME(3) NULL AFTER `lifecycle_status`;

UPDATE `system_i18n`
SET `key` = CASE
  WHEN COALESCE(`key`, '') = '' THEN COALESCE(`locale_key`, '')
  ELSE `key`
END;

ALTER TABLE `system_dict_type`
  ADD COLUMN `module` VARCHAR(64) NOT NULL DEFAULT 'system' AFTER `dict_name`;

ALTER TABLE `system_dict_item`
  ADD COLUMN `item_color` VARCHAR(32) DEFAULT '' AFTER `item_value`;

UPDATE `system_dict_item`
SET `item_color` = CASE
  WHEN COALESCE(`item_color`, '') = '' THEN COALESCE(`tag_type`, '')
  ELSE `item_color`
END;

ALTER TABLE `system_log_oper`
  ADD COLUMN `business_type` INT DEFAULT 0 AFTER `title`,
  ADD COLUMN `oper_name` VARCHAR(64) DEFAULT '' AFTER `method`,
  ADD COLUMN `oper_url` VARCHAR(255) DEFAULT '' AFTER `oper_name`,
  ADD COLUMN `oper_ip` VARCHAR(128) DEFAULT '' AFTER `oper_url`,
  ADD COLUMN `json_result` TEXT AFTER `oper_param`,
  ADD COLUMN `error_msg` TEXT AFTER `status`,
  ADD COLUMN `oper_time` DATETIME(3) NULL AFTER `error_msg`,
  ADD COLUMN `cost_time` BIGINT DEFAULT 0 AFTER `oper_time`;

UPDATE `system_log_oper`
SET
  `oper_name` = CASE
    WHEN COALESCE(`oper_name`, '') = '' THEN COALESCE(`operator_name`, '')
    ELSE `oper_name`
  END,
  `oper_url` = CASE
    WHEN COALESCE(`oper_url`, '') = '' THEN COALESCE(`path`, '')
    ELSE `oper_url`
  END,
  `oper_ip` = CASE
    WHEN COALESCE(`oper_ip`, '') = '' THEN COALESCE(`operator_ip`, '')
    ELSE `oper_ip`
  END,
  `json_result` = CASE
    WHEN COALESCE(`json_result`, '') = '' THEN COALESCE(`result`, '')
    ELSE `json_result`
  END,
  `oper_time` = COALESCE(`oper_time`, `created_at`),
  `cost_time` = CASE
    WHEN COALESCE(`cost_time`, 0) = 0 THEN COALESCE(`cost_ms`, 0)
    ELSE `cost_time`
  END;

ALTER TABLE `system_role_data_scope`
  ADD COLUMN `dept_ids` TEXT NULL AFTER `mode`;

UPDATE `system_role_data_scope`
SET `dept_ids` = CASE
  WHEN COALESCE(`dept_ids`, '') = '' THEN COALESCE(`scope_defs`, '')
  ELSE `dept_ids`
END;
