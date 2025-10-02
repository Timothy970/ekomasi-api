CREATE TABLE IF NOT EXISTS product_discounts (
    product_discount_id char(36) PRIMARY KEY,
    product_id char(36) NOT NULL,
    promotion_type_id bigint unsigned NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_product_discounts
        FOREIGN KEY (product_id) REFERENCES products(product_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_promotion_type
        FOREIGN KEY (promotion_type_id) REFERENCES promotion_types(id)
        ON DELETE CASCADE,
    UNIQUE (product_id, promotion_type_id)
);