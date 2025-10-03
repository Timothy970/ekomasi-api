CREATE TABLE IF NOT EXISTS inventory (
   `inventory_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `variant_id` char(36) NOT NULL,
   `quantity` int NOT NULL DEFAULT '0',
   `low_stock_threshold` int NOT NULL DEFAULT '0',
   `warehouse_id` char(36) DEFAULT NULL,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `last_updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`inventory_id`),
   KEY `idx_inventory_product_id` (`product_id`),
   KEY `inventory_ibfk_2` (`variant_id`),
   CONSTRAINT `inventory_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE,
   CONSTRAINT `inventory_ibfk_2` FOREIGN KEY (`variant_id`) REFERENCES `variants` (`variant_id`) ON DELETE CASCADE,
   CONSTRAINT `inventory_ibfk_3` FOREIGN KEY (`warehouse_id`) REFERENCES `warehouses` (`warehouse_id`) ON DELETE SET NULL
 );