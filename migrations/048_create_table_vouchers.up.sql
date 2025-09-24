CREATE TABLE IF NOT EXISTS vouchers (
    voucher_id      char(36) PRIMARY KEY,
    user_id         char(36) DEFAULT NULL,
    code            VARCHAR(50) UNIQUE NOT NULL,
    balance         DECIMAL(10,2) DEFAULT 0.0, 
    original_value  DECIMAL(10,2) NOT NULL DEFAULT 0.0,     
    expiry_date     DATETIME NOT NULL,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT v_user FOREIGN KEY (user_id) REFERENCES users(user_id)
        ON DELETE CASCADE
);