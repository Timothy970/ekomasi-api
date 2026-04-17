CREATE TABLE IF NOT EXISTS product_variant_combination_options (
    id char(36) PRIMARY KEY,
    combination_id char(36) NOT NULL,
    variant_id char(36) NOT NULL,
    FOREIGN KEY (combination_id) REFERENCES product_variant_combinations(id) ON DELETE CASCADE,
    FOREIGN KEY (variant_id) REFERENCES variants(variant_id) ON DELETE CASCADE
    INDEX idx_combination_id (combination_id),
);