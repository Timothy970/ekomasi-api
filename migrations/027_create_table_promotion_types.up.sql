CREATE TABLE IF NOT EXISTS  `promotion_types` (
   `id` bigint unsigned NOT NULL AUTO_INCREMENT,
   `name` varchar(100) NOT NULL,
   `description` text,
   `value` decimal(10,0) NOT NULL,
   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (`id`),
   UNIQUE KEY `id` (`id`),
   UNIQUE KEY `name` (`name`)
 );