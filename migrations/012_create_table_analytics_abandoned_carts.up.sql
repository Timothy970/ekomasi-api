CREATE TABLE IF NOT EXISTS  `analytics_abandoned_carts` (
   `cart_id` char(36) NOT NULL,
   `user_id` char(36) DEFAULT NULL,
   `guest_notification_details` varchar(255) NOT NULL,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `item_count` int NOT NULL,
   `total_value` decimal(10,2) NOT NULL,
   PRIMARY KEY (`cart_id`),
   KEY `idx_analytics_abandoned_carts_created_at` (`created_at`),
   KEY `idx_analytics_abandoned_carts_user_id` (`user_id`),
   KEY `idx_analytics_abandoned_carts_total_value` (`total_value`),
   CONSTRAINT `analytics_abandoned_carts_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE SET NULL
 );