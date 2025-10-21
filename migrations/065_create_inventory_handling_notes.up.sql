CREATE TABLE IF NOT EXISTS inventory_handling_notes (
    handling_note_id CHAR(36) PRIMARY KEY,
    batch_id CHAR(36) DEFAULT NULL,
    condition_id int DEFAULT NULL,
    handling_notes TEXT DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_handling_notes_batch
        FOREIGN KEY (batch_id) REFERENCES inventory_batches(batch_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_handling_notes_condition
        FOREIGN KEY (condition_id) REFERENCES batch_conditions(condition_id)
        ON DELETE SET NULL
);
