DROP PROCEDURE IF EXISTS add_tenant_col_palette;

CREATE PROCEDURE add_tenant_col_palette(IN col_name VARCHAR(64), IN col_def VARCHAR(255))
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.COLUMNS 
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tenants' AND COLUMN_NAME = col_name
    ) THEN
        SET @s = CONCAT('ALTER TABLE tenants ADD COLUMN ', col_name, ' ', col_def);
        PREPARE stmt FROM @s;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END;

CALL add_tenant_col_palette('app_domain', 'VARCHAR(255) DEFAULT "localhost:3000"');
CALL add_tenant_col_palette('admin_domain', 'VARCHAR(255) DEFAULT "localhost:3001"');
CALL add_tenant_col_palette('app_logo', 'VARCHAR(255) DEFAULT "/images/ekomasi-logo.png"');
CALL add_tenant_col_palette('app_primary_color', 'VARCHAR(50) DEFAULT "#4f46e5"');
CALL add_tenant_col_palette('app_secondary_color', 'VARCHAR(50) DEFAULT "#E8298A"');
CALL add_tenant_col_palette('app_tertiary_color', 'VARCHAR(50) DEFAULT "#EEF2FF"');
CALL add_tenant_col_palette('admin_logo', 'VARCHAR(255) DEFAULT "/images/ekomasi-logo.png"');
CALL add_tenant_col_palette('admin_primary_color', 'VARCHAR(50) DEFAULT "#0f172a"');
CALL add_tenant_col_palette('admin_secondary_color', 'VARCHAR(50) DEFAULT "#AF52DE"');
CALL add_tenant_col_palette('admin_tertiary_color', 'VARCHAR(50) DEFAULT "#1B202E"');

DROP PROCEDURE IF EXISTS add_tenant_col_palette;

UPDATE tenants 
SET 
  app_domain = COALESCE(app_domain, domain, 'localhost:3000'),
  admin_domain = COALESCE(admin_domain, 'localhost:3001'),
  app_logo = COALESCE(app_logo, logo, '/images/ekomasi-logo.png'),
  app_primary_color = COALESCE(app_primary_color, color, '#4f46e5'),
  app_secondary_color = COALESCE(app_secondary_color, '#E8298A'),
  app_tertiary_color = COALESCE(app_tertiary_color, '#EEF2FF'),
  admin_logo = COALESCE(admin_logo, logo, '/images/ekomasi-logo.png'),
  admin_primary_color = COALESCE(admin_primary_color, '#0f172a'),
  admin_secondary_color = COALESCE(admin_secondary_color, '#AF52DE'),
  admin_tertiary_color = COALESCE(admin_tertiary_color, '#1B202E')
WHERE id > 0;
