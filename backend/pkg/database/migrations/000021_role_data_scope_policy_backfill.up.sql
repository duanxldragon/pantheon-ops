-- Preserve the pre-fix effective behavior for existing active roles that have
-- no explicit row: missing policies previously resolved to all. Existing rows
-- (including restricted/custom policies) are never overwritten.
INSERT IGNORE INTO `system_role_data_scope` (`role_key`, `mode`, `dept_ids`)
SELECT r.`role_key`, 'all', ''
FROM `system_role` AS r
LEFT JOIN `system_role_data_scope` AS p ON p.`role_key` = r.`role_key`
WHERE r.`deleted_at` IS NULL
  AND TRIM(r.`role_key`) <> ''
  AND p.`id` IS NULL;
