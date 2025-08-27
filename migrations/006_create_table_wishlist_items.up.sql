CREATE TABLE IF NOT EXISTS  `wishlist_items` (
   `wishlist_item_id` char(36) NOT NULL,
   `wishlist_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   PRIMARY KEY (`wishlist_item_id`),
   KEY `product_id` (`product_id`),
   KEY `idx_wishlist_items_wishlist_id` (`wishlist_id`),
   CONSTRAINT `wishlist_items_ibfk_1` FOREIGN KEY (`wishlist_id`) REFERENCES `wishlists` (`wishlist_id`) ON DELETE CASCADE,
   CONSTRAINT `wishlist_items_ibfk_2` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );