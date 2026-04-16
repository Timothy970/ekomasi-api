CREATE TABLE IF NOT EXISTS  `cart_items` (
   `id` varchar(36) NOT NULL,
   `cart_id` char(36) DEFAULT NULL,
   `product_id` char(36) DEFAULT NULL,
   `quantity` int NOT NULL,
   `variation_sku` varchar(255) DEFAULT NULL,
   `variation_sku_normalized` varchar(255) GENERATED ALWAYS AS (IFNULL(variation_sku, '__NO_VARIANT__')) STORED,
   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (`id`),
   KEY `cart_id` (`cart_id`),
   CONSTRAINT `unique_cart_product_variation` UNIQUE (`cart_id`, `product_id`, `variation_sku_normalized`),
   KEY `cart_items_ibfk_2` (`product_id`),
   CONSTRAINT `cart_items_ibfk_2` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );