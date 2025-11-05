CREATE TABLE IF NOT EXISTS `cart` (
   `cart_id` char(36) NOT NULL,
   `user_id` char(36) DEFAULT NULL,
   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`cart_id`),
   KEY `fk_cart_user` (`user_id`),
   CONSTRAINT `fk_cart_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE
 );