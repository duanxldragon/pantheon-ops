-- Retained down: business_scope_id records who owns each encrypted credential.
-- Dropping the column would make every credential look platform-global again,
-- which is the SEC-004 exposure this migration closes. Roll back by restoring a
-- pre-T05 backup in an isolated environment instead.
SELECT 1;
