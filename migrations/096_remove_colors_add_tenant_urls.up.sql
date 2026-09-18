-- Migration 096: Remove color settings from tenants and introduce tenant_urls table
-- Up Migration

-- Step 1: Create the tenant_urls table
CREATE TABLE IF NOT EXISTS tenant_urls (
    id           INT AUTO_INCREMENT PRIMARY KEY,
    tenant_id    INT NOT NULL,
    url          VARCHAR(255) NOT NULL,
    url_type     ENUM('storefront', 'admin') NOT NULL DEFAULT 'storefront',
    is_primary   TINYINT(1) NOT NULL DEFAULT 0,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_tenant_url (url),
    CONSTRAINT fk_tenant_urls_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Step 2: Migrate existing domain values into tenant_urls

-- Migrate primary storefront domain (domain column)
INSERT IGNORE INTO tenant_urls (tenant_id, url, url_type, is_primary)
SELECT id, domain, 'storefront', 1
FROM tenants
WHERE domain IS NOT NULL AND domain != '';

-- Migrate app_domain if different from domain
INSERT IGNORE INTO tenant_urls (tenant_id, url, url_type, is_primary)
SELECT id, app_domain, 'storefront', 0
FROM tenants
WHERE app_domain IS NOT NULL AND app_domain != '' AND app_domain != domain;

-- Migrate admin_domain
INSERT IGNORE INTO tenant_urls (tenant_id, url, url_type, is_primary)
SELECT id, admin_domain, 'admin', 1
FROM tenants
WHERE admin_domain IS NOT NULL AND admin_domain != '';

-- Step 3: Drop color columns from tenants table

DROP PROCEDURE IF EXISTS drop_col_if_exists;

CREATE PROCEDURE drop_col_if_exists(IN tbl VARCHAR(64), IN col VARCHAR(64))
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND COLUMN_NAME = col
    ) THEN
        SET @s = CONCAT('ALTER TABLE ', tbl, ' DROP COLUMN ', col);
        PREPARE stmt FROM @s;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END;

CALL drop_col_if_exists('tenants', 'color');
CALL drop_col_if_exists('tenants', 'app_color');
CALL drop_col_if_exists('tenants', 'admin_color');
CALL drop_col_if_exists('tenants', 'app_primary_color');
CALL drop_col_if_exists('tenants', 'app_secondary_color');
CALL drop_col_if_exists('tenants', 'app_tertiary_color');
CALL drop_col_if_exists('tenants', 'admin_primary_color');
CALL drop_col_if_exists('tenants', 'admin_secondary_color');
CALL drop_col_if_exists('tenants', 'admin_tertiary_color');
CALL drop_col_if_exists('tenants', 'app_domain');
CALL drop_col_if_exists('tenants', 'admin_domain');

DROP PROCEDURE IF EXISTS drop_col_if_exists;
