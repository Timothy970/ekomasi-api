CREATE TABLE IF NOT EXISTS  `orders` (
   `order_id` char(36) NOT NULL,
   `user_id` char(36) DEFAULT NULL,
   `guest_personal_details` LONGTEXT,
   `guest_delivery_address` LONGTEXT,
   `total_amount` decimal(10,2) NOT NULL,
   `total_discount` decimal(10,2) NOT NULL DEFAULT '0.00',
   `status` varchar(100) NOT NULL DEFAULT 'cart',
   `payment_status` varchar(100) DEFAULT 'PROCESSING'
   `delivery_id` char(36) NOT NULL,
   `is_guest_order` tinyint(1) NOT NULL DEFAULT '0',
   `payment_method` varchar(100) DEFAULT 'MPESA',
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `last_updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`order_id`),
   KEY `delivery_id` (`delivery_id`),
   KEY `idx_orders_user_id` (`user_id`),
   KEY `idx_orders_status` (`status`),
   KEY `idx_orders_created_at` (`created_at`),
   CONSTRAINT `orders_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE SET NULL
 );