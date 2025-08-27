CREATE TABLE IF NOT EXISTS  `product_reviews` (
   `review_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `user_id` char(36) NOT NULL,
   `score` int NOT NULL DEFAULT '10',
   `details` text,
   `status` varchar(100) DEFAULT 'created',
   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`review_id`),
   KEY `user_id` (`user_id`),
   KEY `idx_product_reviews_product_id` (`product_id`),
   CONSTRAINT `product_reviews_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE,
   CONSTRAINT `product_reviews_ibfk_2` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE
 );