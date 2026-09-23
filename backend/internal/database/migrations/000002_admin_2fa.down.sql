-- 000002_admin_2fa.down.sql
ALTER TABLE admins DROP COLUMN IF EXISTS two_factor_secret;
