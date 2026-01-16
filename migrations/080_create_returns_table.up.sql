CREATE TABLE  IF NOT EXISTS `returns` (
   `return_id` char(36) NOT NULL,
   `order_id` char(36) DEFAULT NULL,
   `user_id` char(36) NOT NULL,
   `reason` varchar(255) DEFAULT NULL,
   `status` varchar(50) DEFAULT 'Pending',
   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`return_id`),
   KEY `fk_returns_order` (`order_id`),
   CONSTRAINT `fk_returns_order` FOREIGN KEY (`order_id`) REFERENCES `orders` (`order_id`) ON DELETE CASCADE
 );