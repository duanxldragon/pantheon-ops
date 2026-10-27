-- Re-expose platform.tenant_mode as private on down. The setting seed now
-- writes is_public=1 for fresh installs, so this only affects databases that
-- were upgraded then rolled back.
UPDATE `system_setting`
SET `is_public` = 0
WHERE `setting_key` = 'platform.tenant_mode'
  AND `tenant_id` = 0;
