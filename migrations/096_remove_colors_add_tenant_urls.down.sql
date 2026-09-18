-- Migration 096: Remove color settings from tenants and introduce tenant_urls table
-- Down Migration (Rollback)

-- Step 1: Restore color and domain columns to tenants

DROP PROCEDURE IF EXISTS add_col_if_not_exists;

CREATE PROCEDURE add_col_if_not_exists(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND COLUMN_NAME = col
    ) THEN
        SET @s = CONCAT('ALTER TABLE ', tbl, ' ADD COLUMN ', col, ' ', col_def);
        PREPARE stmt FROM @s;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END;

CALL add_col_if_not_exists('tenants', 'color',                 'VARCHAR(50) DEFAULT NULL');
CALL add_col_if_not_exists('tenants', 'app_color',             'VARCHAR(50) DEFAULT NULL');
CALL add_col_if_not_exists('tenants', 'admin_color',           'VARCHAR(50) DEFAULT NULL');
CALL add_col_if_not_exists('tenants', 'app_primary_color',     'VARCHAR(50) DEFAULT "#4f46e5"');
CALL add_col_if_not_exists('tenants', 'app_secondary_color',   'VARCHAR(50) DEFAULT "#E8298A"');
CALL add_col_if_not_exists('tenants', 'app_tertiary_color',    'VARCHAR(50) DEFAULT "#EEF2FF"');
CALL add_col_if_not_exists('tenants', 'admin_primary_color',   'VARCHAR(50) DEFAULT "#0f172a"');
CALL add_col_if_not_exists('tenants', 'admin_secondary_color', 'VARCHAR(50) DEFAULT "#AF52DE"');
CALL add_col_if_not_exists('tenants', 'admin_tertiary_color',  'VARCHAR(50) DEFAULT "#1B202E"');
CALL add_col_if_not_exists('tenants', 'app_domain',            'VARCHAR(255) DEFAULT NULL');
CALL add_col_if_not_exists('tenants', 'admin_domain',          'VARCHAR(255) DEFAULT NULL');

DROP PROCEDURE IF EXISTS add_col_if_not_exists;

-- Step 2: Restore app_domain and admin_domain from tenant_urls
UPDATE tenants t
JOIN tenant_urls u ON u.tenant_id = t.id AND u.url_type = 'storefront' AND u.is_primary = 1
SET t.app_domain = u.url;

UPDATE tenants t
JOIN tenant_urls u ON u.tenant_id = t.id AND u.url_type = 'admin' AND u.is_primary = 1
SET t.admin_domain = u.url;

-- Step 3: Drop the tenant_urls table
DROP TABLE IF EXISTS tenant_urls;
