CREATE TABLE IF NOT EXISTS voucher_orders (
    voucher_order_id  CHAR(36) PRIMARY KEY,
    voucher_id        CHAR(36) NOT NULL,
    amount            DECIMAL(10, 2) NOT NULL,
    status            ENUM('PENDING', 'COMPLETED', 'FAILED') DEFAULT 'PENDING',
    payment_method    VARCHAR(50) NOT NULL,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    CONSTRAINT fk_voucher_order FOREIGN KEY (voucher_id) REFERENCES vouchers(voucher_id) ON DELETE CASCADE
);