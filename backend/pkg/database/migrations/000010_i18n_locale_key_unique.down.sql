-- Deduplicated rows are not restorable; only drop the index when present.
SET @i18n_unique_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n'
    AND index_name = 'uidx_system_i18n_locale_key'
);
SET @i18n_drop_stmt := IF(
  @i18n_unique_idx > 0,
  'DROP INDEX uidx_system_i18n_locale_key ON system_i18n',
  'SELECT 1'
);
PREPARE i18n_drop_stmt FROM @i18n_drop_stmt;
EXECUTE i18n_drop_stmt;
DEALLOCATE PREPARE i18n_drop_stmt;
