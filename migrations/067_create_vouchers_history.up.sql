CREATE TABLE IF NOT EXISTS vouchers_history (
    history_id CHAR(36) PRIMARY KEY,
    voucher_id CHAR(36) NOT NULL,
    redeemed_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    amount_redeemed DECIMAL(10,2) NOT NULL,
    items_log LONGTEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_voucher
        FOREIGN KEY (voucher_id) REFERENCES vouchers(voucher_id)
        ON DELETE CASCADE
);
