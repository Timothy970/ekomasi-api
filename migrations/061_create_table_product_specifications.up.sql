CREATE TABLE IF NOT EXISTS product_specifications (
    specifications_id CHAR(36) PRIMARY KEY,
    product_id CHAR(36) NOT NULL,
    weight DECIMAL(10,2) DEFAULT NULL,
    weight_limit DECIMAL(10,2) DEFAULT NULL,
    dimensions VARCHAR(255) DEFAULT NULL,
    manufacturer VARCHAR(255) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_product_specifications_product
        FOREIGN KEY (product_id) REFERENCES products(product_id)
        ON DELETE CASCADE
);
