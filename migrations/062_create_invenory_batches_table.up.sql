CREATE TABLE IF NOT EXISTS inventory_batches (
    batch_id CHAR(36) PRIMARY KEY,
    inventory_id CHAR(36) NOT NULL,
    batch_number VARCHAR(100) NOT NULL,
    images JSON DEFAULT NULL,
    expiry_date DATE DEFAULT NULL,
    manufacturing_date DATE DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_inventory_batches_inventory
        FOREIGN KEY (inventory_id) REFERENCES inventory(inventory_id)
        ON DELETE CASCADE
);
