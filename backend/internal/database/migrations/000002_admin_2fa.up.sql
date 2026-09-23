-- 000002_admin_2fa.up.sql
ALTER TABLE admins ADD COLUMN IF NOT EXISTS two_factor_secret text;
