CREATE TABLE IF NOT EXISTS transactions (
    transaction_id CHAR(36) NOT NULL,     
    order_id       CHAR(36) DEFAULT NULL,   
    mpesa_reference VARCHAR(100) DEFAULT NULL,
    transaction_reference VARCHAR(100) DEFAULT NULL,
    phone_number   VARCHAR(20) DEFAULT NULL,
    account_number VARCHAR(50) DEFAULT NULL,
    amount         DECIMAL(10, 2) NOT NULL,    
    payment_method VARCHAR(50) NOT NULL,     
    status         VARCHAR(50) DEFAULT 'Pending',
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (transaction_id),
    CONSTRAINT fk_transactions_order FOREIGN KEY (order_id) REFERENCES orders(order_id) ON DELETE CASCADE
);