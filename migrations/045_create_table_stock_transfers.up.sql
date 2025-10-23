CREATE TABLE IF NOT EXISTS stock_transfers (
   `transfer_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `variant_id` char(36) DEFAULT NULL,
   `from_warehouse_id` char(36) NOT NULL,
   `to_warehouse_id` char(36) NOT NULL,
   `quantity` int NOT NULL DEFAULT '0',
   `transfer_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `transfer_details` text,
   PRIMARY KEY (`transfer_id`),
   KEY `variant_id` (`variant_id`),
   KEY `from_warehouse_id` (`from_warehouse_id`),
   KEY `to_warehouse_id` (`to_warehouse_id`),
   KEY `idx_stock_transfers_product_id` (`product_id`),
   CONSTRAINT `stock_transfers_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE RESTRICT,
   CONSTRAINT `stock_transfers_ibfk_2` FOREIGN KEY (`variant_id`) REFERENCES `variants` (`variant_id`) ON DELETE RESTRICT,
   CONSTRAINT `stock_transfers_ibfk_3` FOREIGN KEY (`from_warehouse_id`) REFERENCES `warehouses` (`warehouse_id`) ON DELETE RESTRICT,
   CONSTRAINT `stock_transfers_ibfk_4` FOREIGN KEY (`to_warehouse_id`) REFERENCES `warehouses` (`warehouse_id`) ON DELETE RESTRICT
 );