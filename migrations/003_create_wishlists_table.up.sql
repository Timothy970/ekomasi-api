CREATE TABLE IF NOT EXISTS `wishlists` (
   `wishlist_id` char(36) NOT NULL,
   `user_id` char(36) NOT NULL,
   `name` varchar(100) NOT NULL,
   `is_public` tinyint(1) NOT NULL DEFAULT '0',
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `last_updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`wishlist_id`),
   KEY `idx_wishlists_user_id` (`user_id`),
   CONSTRAINT `wishlists_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE
 );