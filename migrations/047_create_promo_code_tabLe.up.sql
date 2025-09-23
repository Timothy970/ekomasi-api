CREATE TABLE IF NOT EXISTS promo_codes (
    promo_code_id   char(36) PRIMARY KEY,
    code            VARCHAR(100) UNIQUE NOT NULL,
    description     TEXT,
    discount_type   ENUM('PERCENTAGE', 'FIXED') NOT NULL DEFAULT "FIXED",
    discount_value  DECIMAL(10,2) NOT NULL DEFAULT 0,
    expires_at      DATETIME NOT NULL,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);