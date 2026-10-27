-- Operator attribution for dynamic i18n writes (fix-report §4 item 5, P3 governance,
-- task 2026-09-24-fix-report-residual-closeout).
--
-- system_i18n previously had no operator trace: the operation log held the actor,
-- but the row itself could not answer "who last edited this translation". The
-- runtime now writes updated_by on Create/Update/Import/SyncMissingKeys.
--
-- Guarded via information_schema + PREPARE (same compat pattern as 000016/000017/
-- 000018) so it is safe on re-run, on bootstrapped/partial schemas where
-- system_i18n may not exist yet, and on dev databases whose GORM AutoMigrate
-- path (PANTHEON_AUTO_MIGRATE=true) already added the column.
--
-- Column is NULLable: legacy rows predate attribution and the zero value of the
-- Go model string ('') is written explicitly by the runtime.

SET @t := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n'
);
SET @c := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n'
    AND column_name = 'updated_by'
);
SET @s := IF(
  @t > 0 AND @c = 0,
  'ALTER TABLE `system_i18n` ADD COLUMN `updated_by` VARCHAR(64) NULL AFTER `updated_at`',
  'SELECT 1'
);
PREPARE x FROM @s;
EXECUTE x;
DEALLOCATE PREPARE x;
