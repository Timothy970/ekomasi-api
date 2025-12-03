CREATE TABLE IF NOT EXISTS  `bundle_products` (
   `bundle_product_id` char(36) NOT NULL,
   `bundle_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `quantity` int NOT NULL DEFAULT '1',
   PRIMARY KEY (`bundle_product_id`),
   KEY `product_id` (`product_id`),
   KEY `idx_bundle_products_bundle_id` (`bundle_id`),
   CONSTRAINT `bundle_products_ibfk_1` FOREIGN KEY (`bundle_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE,
   CONSTRAINT `bundle_products_ibfk_2` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );