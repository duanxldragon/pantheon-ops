-- Rollback of 000019 up: drop system_i18n.updated_by when present.
-- Guarded so a database that never received the column (or already rolled back)
-- stays a no-op instead of failing the rollback.

SET @c := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n'
    AND column_name = 'updated_by'
);
SET @s := IF(
  @c > 0,
  'ALTER TABLE `system_i18n` DROP COLUMN `updated_by`',
  'SELECT 1'
);
PREPARE x FROM @s;
EXECUTE x;
DEALLOCATE PREPARE x;
