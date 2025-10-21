CREATE TABLE IF NOT EXISTS deal_products (
    product_deal_id char(36) PRIMARY KEY,
    deal_id char(36) NOT NULL,
    product_id char(36) NOT NULL,
    discount DECIMAL(10, 2) DEFAULT NULL,
    discount_type varchar(100) DEFAULT 'fixed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_deal
        FOREIGN KEY (deal_id) REFERENCES deals(deal_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_product_deals
        FOREIGN KEY (product_id) REFERENCES products(product_id)
        ON DELETE CASCADE,
    UNIQUE (deal_id, product_id)
);