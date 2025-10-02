CREATE TABLE IF NOT EXISTS product_charges (
    product_charge_id char(36) PRIMARY KEY,
    product_id char(36) NOT NULL,
    charge_id char(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_product_id
        FOREIGN KEY (product_id) REFERENCES products(product_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_charge
        FOREIGN KEY (charge_id) REFERENCES charges(charge_id)
        ON DELETE CASCADE,
    UNIQUE (product_id, charge_id)
);