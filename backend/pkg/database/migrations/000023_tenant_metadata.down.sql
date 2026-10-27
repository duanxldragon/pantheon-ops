-- Reverses only the schema addition from 000023_tenant_metadata.up.sql.
--
-- The placeholder id normalization (id <> 0 -> 0) is deliberately NOT reverted:
-- it is a correctness repair mandated by TENANT_CONTRACT_V1 section 2.3, not a
-- feature. Moving the row back to a non-zero id would re-expose the reserved
-- platform/global placeholder as an ordinary, deletable tenant in the console,
-- which is the defect this migration exists to remove.

SET @tenants_metadata_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'tenants'
    AND column_name = 'metadata'
);

SET @tenants_drop_metadata_stmt := IF(
  @tenants_metadata_col > 0,
  'ALTER TABLE `tenants` DROP COLUMN `metadata`',
  'SELECT 1'
);
PREPARE tenants_drop_metadata_stmt FROM @tenants_drop_metadata_stmt;
EXECUTE tenants_drop_metadata_stmt;
DEALLOCATE PREPARE tenants_drop_metadata_stmt;
