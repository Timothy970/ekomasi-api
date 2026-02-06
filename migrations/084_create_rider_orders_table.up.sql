CREATE TABLE IF NOT EXISTS rider_orders (
    rider_order_id CHAR(36) PRIMARY KEY,
    rider_id CHAR(36) NOT NULL,
    order_id CHAR(36) NOT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (rider_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (order_id) REFERENCES orders(order_id) ON DELETE CASCADE,
    UNIQUE (rider_id, order_id)
);