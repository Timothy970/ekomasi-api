CREATE TABLE IF NOT EXISTS promocodes (
    promo_code_id  CHAR(36) PRIMARY KEY,
    code           VARCHAR(50) NOT NULL UNIQUE,
    description    TEXT,
    discount_type  ENUM('PERCENTAGE', 'FIXED') NOT NULL DEFAULT 'PERCENTAGE',
    discount_value DECIMAL(10, 2) NOT NULL,
    expires_at     TIMESTAMP NOT NULL,
    is_active      BOOLEAN DEFAULT TRUE,
    minimum_order_value DECIMAL(10, 2),
    maximum_use    INT DEFAULT 1,
    times_used     INT DEFAULT 0,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);