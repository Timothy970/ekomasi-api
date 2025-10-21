CREATE TABLE IF NOT EXISTS warranty_types (
    warranty_type_id   CHAR(36) PRIMARY KEY,
    name               VARCHAR(100) NOT NULL,
    description        TEXT NULL              
);
