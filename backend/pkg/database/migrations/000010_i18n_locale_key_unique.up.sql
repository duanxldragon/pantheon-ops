-- Enforce i18n resource uniqueness on (locale, key) for upgraded legacy databases.
-- Fresh installs already get this index from database/system_init.sql, and the i18n
-- module Bootstrap creates it at runtime; this migration makes the guarantee part of
-- the versioned schema so upgrades no longer depend on application startup order.
-- Guarded via information_schema (same compat pattern as 000005-000007) so it is
-- safe on bootstrapped/partial schemas where the table or index may already differ.

SET @i18n_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n'
);
SET @i18n_key_column_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n'
    AND column_name = 'key'
);
SET @i18n_unique_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n'
    AND index_name = 'uidx_system_i18n_locale_key'
);

-- Step 1: deduplicate — keep the newest row (highest id) per (locale, key),
-- matching the "latest write wins" semantics of the runtime normalizer.
SET @i18n_dedupe_stmt := IF(
  @i18n_table_exists > 0 AND @i18n_key_column_exists > 0 AND @i18n_unique_idx = 0,
  'DELETE older FROM system_i18n older JOIN system_i18n newer ON older.locale = newer.locale AND older.`key` = newer.`key` AND older.id < newer.id',
  'SELECT 1'
);
PREPARE i18n_dedupe_stmt FROM @i18n_dedupe_stmt;
EXECUTE i18n_dedupe_stmt;
DEALLOCATE PREPARE i18n_dedupe_stmt;

-- Step 2: add the unique index only when absent (runtime Bootstrap may have
-- created it already on databases that ran an older binary).
SET @i18n_index_stmt := IF(
  @i18n_table_exists > 0 AND @i18n_key_column_exists > 0 AND @i18n_unique_idx = 0,
  'CREATE UNIQUE INDEX uidx_system_i18n_locale_key ON system_i18n (locale, `key`)',
  'SELECT 1'
);
PREPARE i18n_index_stmt FROM @i18n_index_stmt;
EXECUTE i18n_index_stmt;
DEALLOCATE PREPARE i18n_index_stmt;
