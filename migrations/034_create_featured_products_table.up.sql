CREATE TABLE IF NOT EXISTS featured_products (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    product_id char(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(product_id) ON DELETE CASCADE
);
