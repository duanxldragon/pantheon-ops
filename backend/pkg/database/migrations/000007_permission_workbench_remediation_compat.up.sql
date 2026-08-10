SET @permission_remediation_issue_key_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND column_name = 'issue_key'
);
SET @permission_remediation_issue_key_stmt := IF(
  @permission_remediation_issue_key_exists = 0,
  'ALTER TABLE `permission_workbench_remediation_event` ADD COLUMN `issue_key` VARCHAR(255) NOT NULL DEFAULT '''' AFTER `issue_type`',
  'SELECT 1'
);
PREPARE permission_remediation_issue_key_stmt FROM @permission_remediation_issue_key_stmt;
EXECUTE permission_remediation_issue_key_stmt;
DEALLOCATE PREPARE permission_remediation_issue_key_stmt;

SET @permission_remediation_before_state_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND column_name = 'before_state'
);
SET @permission_remediation_before_state_stmt := IF(
  @permission_remediation_before_state_exists = 0,
  'ALTER TABLE `permission_workbench_remediation_event` ADD COLUMN `before_state` VARCHAR(32) NOT NULL DEFAULT ''unknown'' AFTER `issue_key`',
  'SELECT 1'
);
PREPARE permission_remediation_before_state_stmt FROM @permission_remediation_before_state_stmt;
EXECUTE permission_remediation_before_state_stmt;
DEALLOCATE PREPARE permission_remediation_before_state_stmt;

SET @permission_remediation_after_state_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND column_name = 'after_state'
);
SET @permission_remediation_after_state_stmt := IF(
  @permission_remediation_after_state_exists = 0,
  'ALTER TABLE `permission_workbench_remediation_event` ADD COLUMN `after_state` VARCHAR(32) NOT NULL DEFAULT ''unknown'' AFTER `before_state`',
  'SELECT 1'
);
PREPARE permission_remediation_after_state_stmt FROM @permission_remediation_after_state_stmt;
EXECUTE permission_remediation_after_state_stmt;
DEALLOCATE PREPARE permission_remediation_after_state_stmt;

SET @permission_remediation_action_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND column_name = 'action'
);
SET @permission_remediation_action_stmt := IF(
  @permission_remediation_action_exists = 0,
  'ALTER TABLE `permission_workbench_remediation_event` ADD COLUMN `action` VARCHAR(32) NOT NULL DEFAULT ''legacy'' AFTER `after_state`',
  'SELECT 1'
);
PREPARE permission_remediation_action_stmt FROM @permission_remediation_action_stmt;
EXECUTE permission_remediation_action_stmt;
DEALLOCATE PREPARE permission_remediation_action_stmt;

SET @permission_remediation_created_count_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND column_name = 'created_count'
);
SET @permission_remediation_created_count_stmt := IF(
  @permission_remediation_created_count_exists = 0,
  'ALTER TABLE `permission_workbench_remediation_event` ADD COLUMN `created_count` INT DEFAULT 0 AFTER `action`',
  'SELECT 1'
);
PREPARE permission_remediation_created_count_stmt FROM @permission_remediation_created_count_stmt;
EXECUTE permission_remediation_created_count_stmt;
DEALLOCATE PREPARE permission_remediation_created_count_stmt;

SET @permission_remediation_skipped_count_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND column_name = 'skipped_count'
);
SET @permission_remediation_skipped_count_stmt := IF(
  @permission_remediation_skipped_count_exists = 0,
  'ALTER TABLE `permission_workbench_remediation_event` ADD COLUMN `skipped_count` INT DEFAULT 0 AFTER `created_count`',
  'SELECT 1'
);
PREPARE permission_remediation_skipped_count_stmt FROM @permission_remediation_skipped_count_stmt;
EXECUTE permission_remediation_skipped_count_stmt;
DEALLOCATE PREPARE permission_remediation_skipped_count_stmt;

SET @permission_remediation_action_index_exists := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND index_name = 'idx_permission_remediation_action'
);
SET @permission_remediation_action_index_stmt := IF(
  @permission_remediation_action_index_exists = 0,
  'ALTER TABLE `permission_workbench_remediation_event` ADD INDEX `idx_permission_remediation_action` (`action`)',
  'SELECT 1'
);
PREPARE permission_remediation_action_index_stmt FROM @permission_remediation_action_index_stmt;
EXECUTE permission_remediation_action_index_stmt;
DEALLOCATE PREPARE permission_remediation_action_index_stmt;

SET @permission_remediation_detail_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND column_name = 'detail'
);
SET @permission_remediation_remediated_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_workbench_remediation_event'
    AND column_name = 'remediated'
);
SET @permission_remediation_backfill_stmt := IF(
  @permission_remediation_detail_exists = 1 AND @permission_remediation_remediated_exists = 1,
  'UPDATE `permission_workbench_remediation_event`
   SET
     `issue_key` = CASE
       WHEN COALESCE(`issue_key`, '''') = '''' THEN COALESCE(`detail`, '''')
       ELSE `issue_key`
     END,
     `before_state` = CASE
       WHEN COALESCE(`before_state`, '''') <> '''' THEN `before_state`
       WHEN COALESCE(`remediated`, 0) = 1 THEN ''api-gap''
       ELSE ''unknown''
     END,
     `after_state` = CASE
       WHEN COALESCE(`after_state`, '''') <> '''' THEN `after_state`
       WHEN COALESCE(`remediated`, 0) = 1 THEN ''complete''
       ELSE ''unknown''
     END,
     `action` = CASE
       WHEN COALESCE(`action`, '''') <> '''' THEN `action`
       WHEN COALESCE(`remediated`, 0) = 1 THEN ''remediated''
       ELSE ''legacy''
     END,
     `created_count` = COALESCE(`created_count`, 0),
     `skipped_count` = COALESCE(`skipped_count`, 0)',
  'SELECT 1'
);
PREPARE permission_remediation_backfill_stmt FROM @permission_remediation_backfill_stmt;
EXECUTE permission_remediation_backfill_stmt;
DEALLOCATE PREPARE permission_remediation_backfill_stmt;
