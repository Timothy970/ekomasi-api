CREATE TABLE IF NOT EXISTS returns (
    return_id      CHAR(36) NOT NULL,           -- unique ID for the return
    order_id       CHAR(36) DEFAULT NULL,           -- link back to original order
    product_id     CHAR(36) DEFAULT NULL,           -- product being returned
    quantity       INT DEFAULT 1,      -- number of items returned
    reason         VARCHAR(255),                -- optional reason for return
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (return_id),
    CONSTRAINT fk_returns_order FOREIGN KEY (order_id) REFERENCES orders(order_id) ON DELETE CASCADE,
    CONSTRAINT fk_returns_product FOREIGN KEY (product_id) REFERENCES products(product_id) ON DELETE CASCADE
);
