CREATE TABLE IF NOT EXISTS batch_inspections (
    inspection_id CHAR(36) PRIMARY KEY,
    batch_id CHAR(36) DEFAULT NULL,
    inspection_date DATE NOT NULL,
    inspector_id CHAR(36) DEFAULT NULL,
    inspection_notes TEXT DEFAULT NULL,
    images JSON DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_batch_inspections_batch
        FOREIGN KEY (batch_id) REFERENCES inventory_batches(batch_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_batch_inspections_inspector
        FOREIGN KEY (inspector_id) REFERENCES users(user_id)
        ON DELETE SET NULL
);
