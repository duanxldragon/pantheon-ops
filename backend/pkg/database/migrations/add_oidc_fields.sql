-- Add OIDC fields to system_user table
-- Migration: Add SSO/OIDC support

ALTER TABLE system_user
ADD COLUMN oidc_subject VARCHAR(255) NULL COMMENT 'OIDC sub claim (unique user ID from provider)',
ADD COLUMN oidc_provider VARCHAR(100) NULL COMMENT 'OIDC provider identifier',
ADD COLUMN oidc_last_sync DATETIME NULL COMMENT 'Last time profile was synced from provider',
ADD COLUMN auth_type VARCHAR(20) DEFAULT 'local' COMMENT 'Authentication type: local or oidc';

-- Add unique index for OIDC subject
CREATE UNIQUE INDEX idx_system_user_oidc_subject ON system_user(oidc_subject);

-- Add index for auth_type
CREATE INDEX idx_system_user_auth_type ON system_user(auth_type);
