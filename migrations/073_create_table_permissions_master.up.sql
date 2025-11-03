CREATE TABLE IF NOT EXISTS permissions_master (
    permission_master_id CHAR(36) PRIMARY KEY,
    category VARCHAR(100) NOT NULL,
    permission_key VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE(category, permission_key)
);