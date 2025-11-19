CREATE TABLE IF NOT EXISTS return_products (
    return_product_id CHAR(36) NOT NULL,           -- unique ID for the return product entry
    return_id         CHAR(36) NOT NULL,           -- link back to the return
    product_id       CHAR(36) NOT NULL,           -- product being returned
    quantity         INT DEFAULT 1,      -- number of items returned
    PRIMARY KEY (return_product_id),
    CONSTRAINT fk_return_products_return FOREIGN KEY (return_id) REFERENCES returns(return_id) ON DELETE CASCADE,
    CONSTRAINT fk_return_products_product FOREIGN KEY (product_id) REFERENCES products(product_id) ON DELETE CASCADE
);