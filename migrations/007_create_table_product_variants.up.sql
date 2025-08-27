CREATE TABLE IF NOT EXISTS  `product_variants` (
   `variant_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `name` varchar(100) NOT NULL,
   `additional_price` decimal(10,2) NOT NULL DEFAULT '0.00',
   `stock_quantity` int NOT NULL DEFAULT '0',
   PRIMARY KEY (`variant_id`),
   KEY `idx_product_variants_product_id` (`product_id`),
   CONSTRAINT `product_variants_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );