ALTER TABLE cart DROP FOREIGN KEY fk_cart_tenant;
ALTER TABLE cart DROP COLUMN tenant_id;

ALTER TABLE orders DROP FOREIGN KEY fk_orders_tenant;
ALTER TABLE orders DROP COLUMN tenant_id;

ALTER TABLE products DROP FOREIGN KEY fk_products_tenant;
ALTER TABLE products DROP COLUMN tenant_id;

ALTER TABLE categories DROP FOREIGN KEY fk_categories_tenant;
ALTER TABLE categories DROP COLUMN tenant_id;

ALTER TABLE users DROP FOREIGN KEY fk_users_tenant;
ALTER TABLE users DROP COLUMN tenant_id;

DROP TABLE IF EXISTS tenants;
