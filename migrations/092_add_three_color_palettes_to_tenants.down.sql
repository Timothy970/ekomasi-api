ALTER TABLE tenants 
DROP COLUMN IF EXISTS app_domain,
DROP COLUMN IF EXISTS admin_domain,
DROP COLUMN IF EXISTS app_logo,
DROP COLUMN IF EXISTS app_primary_color,
DROP COLUMN IF EXISTS app_secondary_color,
DROP COLUMN IF EXISTS app_tertiary_color,
DROP COLUMN IF EXISTS admin_logo,
DROP COLUMN IF EXISTS admin_primary_color,
DROP COLUMN IF EXISTS admin_secondary_color,
DROP COLUMN IF EXISTS admin_tertiary_color;
