CREATE TABLE IF NOT EXISTS  `banners` (
   `id` int NOT NULL AUTO_INCREMENT,
   `image_url` varchar(255) NOT NULL,
   `type` varchar(100) DEFAULT 'banners',
   `text` text,
   `heading` varchar(255) DEFAULT NULL,
   `button_text` varchar(50) DEFAULT NULL,
   `button_url` varchar(255) DEFAULT NULL,
   `display_order` int NOT NULL DEFAULT '0',
   `is_active` tinyint(1) NOT NULL DEFAULT '1',
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`id`),
   KEY `idx_display_order` (`display_order`),
   KEY `idx_is_active` (`is_active`)
 );