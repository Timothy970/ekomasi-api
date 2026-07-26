ALTER TABLE tenants 
ADD COLUMN app_domain VARCHAR(255) DEFAULT 'localhost:3000',
ADD COLUMN admin_domain VARCHAR(255) DEFAULT 'localhost:3001',
ADD COLUMN app_logo VARCHAR(255) DEFAULT '/images/ekomasi-logo.png',
ADD COLUMN app_color VARCHAR(50) DEFAULT '#4f46e5',
ADD COLUMN admin_logo VARCHAR(255) DEFAULT '/images/ekomasi-logo.png',
ADD COLUMN admin_color VARCHAR(50) DEFAULT '#0f172a';

UPDATE tenants 
SET 
  app_domain = COALESCE(app_domain, domain, 'localhost:3000'),
  admin_domain = COALESCE(admin_domain, 'localhost:3001'),
  app_logo = COALESCE(app_logo, logo, '/images/ekomasi-logo.png'),
  app_color = COALESCE(app_color, color, '#4f46e5'),
  admin_logo = COALESCE(admin_logo, logo, '/images/ekomasi-logo.png'),
  admin_color = COALESCE(admin_color, '#0f172a')
WHERE id > 0;
