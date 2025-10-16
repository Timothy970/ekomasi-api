CREATE TABLE IF NOT EXISTS voucher_purchases (
    purchase_id       CHAR(36) PRIMARY KEY,
    voucher_id        CHAR(36) NOT NULL,
    from_user_id      CHAR(36) NULL,
    from_name         VARCHAR(255) NOT NULL,
    to_name           VARCHAR(255) NOT NULL,
    to_email          VARCHAR(255) NOT NULL,
    personalized_msg  LONGTEXT,
    delivery_time     DATETIME,
    sent_at           DATETIME NULL,
    status            ENUM('PENDING', 'SENT', 'FAILED') DEFAULT 'PENDING',
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    CONSTRAINT fk_voucher_purchase FOREIGN KEY (voucher_id) REFERENCES vouchers(voucher_id)
);
