CREATE TABLE IF NOT EXISTS product_variants (
   `variant_id` char(36) NOT NULL,
   `product_variants_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `additional_price` decimal(10,2) NOT NULL DEFAULT '0.00',
   `stock_quantity` int DEFAULT '0',
   PRIMARY KEY (`product_variants_id`),
   UNIQUE KEY `uq_variant_product` (`variant_id`,`product_id`),
   KEY `idx_product_variants_product_id` (`product_id`),
   KEY `fk_product_variants_variant_product` (`variant_id`,`product_id`),
   CONSTRAINT `fk_product_variants_variant` FOREIGN KEY (`variant_id`) REFERENCES `variants` (`variant_id`) ON DELETE CASCADE ON UPDATE CASCADE,
   CONSTRAINT `product_variants_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );