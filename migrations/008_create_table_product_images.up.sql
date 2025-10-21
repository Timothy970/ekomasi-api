CREATE TABLE IF NOT EXISTS  `product_images` (
   `image_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `url` longtext,
   `is_primary` tinyint(1) NOT NULL DEFAULT '0',
   `type` varchar(50) NOT NULL DEFAULT 'gallery',
   PRIMARY KEY (`image_id`),
   KEY `idx_product_images_product_id` (`product_id`),
   CONSTRAINT `product_images_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );