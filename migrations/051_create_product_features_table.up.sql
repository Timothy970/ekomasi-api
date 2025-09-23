CREATE TABLE IF NOT EXISTS product_features (
    feature_id      BIGINT AUTO_INCREMENT PRIMARY KEY,
    product_id      BIGINT NOT NULL,
    header          TEXT,
    description     LONGTEXT,
    image           VARCHAR(255)
    image_position  ENUM('left','right','top','bottom') DEFAULT 'left',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,    
    CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(product_id)
        ON DELETE CASCADE
);
