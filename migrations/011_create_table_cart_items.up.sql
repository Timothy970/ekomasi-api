CREATE TABLE IF NOT EXISTS  `cart_items` (
   `id` varchar(36) NOT NULL,
   `user_id` char(36) DEFAULT NULL,
   `product_id` char(36) DEFAULT NULL,
   `quantity` int NOT NULL,
   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (`id`),
   KEY `cart_id` (`user_id`),
   CONSTRAINT `unique_user_product` UNIQUE (`user_id`, `product_id`),
   KEY `cart_items_ibfk_2` (`product_id`),
   CONSTRAINT `cart_items_ibfk_2` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );