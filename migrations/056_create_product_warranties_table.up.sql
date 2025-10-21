CREATE TABLE IF NOT EXISTS product_warranties (
    warranty_id        CHAR(36) PRIMARY KEY,
    product_id         CHAR(36) NOT NULL,
    warranty_type_id   CHAR(36) NOT NULL,      -- FK to warranty_types
    warranty_period    INT NOT NULL,           -- in months
    manufacturing_date DATE NOT NULL,
    expiry_date        DATE NOT NULL,
    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(product_id),
    FOREIGN KEY (warranty_type_id) REFERENCES warranty_types(warranty_type_id)

);
