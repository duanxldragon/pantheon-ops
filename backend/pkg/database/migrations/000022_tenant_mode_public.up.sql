-- The tenant console menu is gated on platform.tenant_mode in the frontend
-- (same capability pattern as org.enabled). Fresh installs get is_public=1
-- from the setting seed; this data migration flips existing rows so already
-- provisioned databases expose the flag too.
UPDATE `system_setting`
SET `is_public` = 1
WHERE `setting_key` = 'platform.tenant_mode'
  AND `tenant_id` = 0
  AND `is_public` <> 1;
