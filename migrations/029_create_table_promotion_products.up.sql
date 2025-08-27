CREATE TABLE IF NOT EXISTS  `promotion_products` (
   `promotion_product_id` char(36) NOT NULL,
   `promotion_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `discount_percentage` decimal(5,2) NOT NULL,
   PRIMARY KEY (`promotion_product_id`),
   KEY `idx_promotion_products_promotion_id` (`promotion_id`),
   KEY `idx_promotion_products_product_id` (`product_id`),
   CONSTRAINT `promotion_products_ibfk_1` FOREIGN KEY (`promotion_id`) REFERENCES `promotions` (`promotion_id`) ON DELETE CASCADE,
   CONSTRAINT `promotion_products_ibfk_2` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );