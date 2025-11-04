CREATE TABLE IF NOT EXISTS `user_addresses` (
   `address_id` char(50) NOT NULL,
   `user_id` char(50) NOT NULL,
   `address` varchar(255) NOT NULL,
   `country` varchar(255) DEFAULT NULL,
   `apartment` varchar(255) DEFAULT NULL,
   `city` varchar(255) DEFAULT NULL,
   `zip_code` varchar(255) DEFAULT NULL,
   `is_default` tinyint(1) NOT NULL DEFAULT '0',
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`address_id`),
   KEY `idx_user_addresses_user_id` (`user_id`),
   CONSTRAINT `user_addresses_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE
 );