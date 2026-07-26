CREATE TABLE IF NOT EXISTS tenants (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL UNIQUE,
    slogan VARCHAR(255),
    logo VARCHAR(255),
    color VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO tenants (id, name, domain, slogan, logo, color)
VALUES (1, 'Ekomasi Store', 'localhost:3000', 'Your premium e-commerce platform', '/images/ekomasi-logo.png', '#4f46e5')
ON DUPLICATE KEY UPDATE name=name;

DROP PROCEDURE IF EXISTS add_tenant_id_col;

CREATE PROCEDURE add_tenant_id_col(IN tbl VARCHAR(64), IN fk VARCHAR(64))
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.COLUMNS 
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND COLUMN_NAME = 'tenant_id'
    ) THEN
        SET @s1 = CONCAT('ALTER TABLE ', tbl, ' ADD COLUMN tenant_id INT DEFAULT 1');
        PREPARE stmt1 FROM @s1;
        EXECUTE stmt1;
        DEALLOCATE PREPARE stmt1;

        SET @s2 = CONCAT('ALTER TABLE ', tbl, ' ADD CONSTRAINT ', fk, ' FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE SET NULL');
        PREPARE stmt2 FROM @s2;
        EXECUTE stmt2;
        DEALLOCATE PREPARE stmt2;
    END IF;
END;

CALL add_tenant_id_col('users', 'fk_users_tenant');
CALL add_tenant_id_col('categories', 'fk_categories_tenant');
CALL add_tenant_id_col('products', 'fk_products_tenant');
CALL add_tenant_id_col('orders', 'fk_orders_tenant');
CALL add_tenant_id_col('cart', 'fk_cart_tenant');

DROP PROCEDURE IF EXISTS add_tenant_id_col;
