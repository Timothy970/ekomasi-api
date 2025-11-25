CREATE TABLE IF NOT EXISTS  `categories` (
   `category_id` char(36) NOT NULL,
   `name` varchar(100) NOT NULL,
   `parent_category_id` char(36) DEFAULT NULL,
   `image` varchar(255) DEFAULT NULL,
   `description` longtext DEFAULT NULL,
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`category_id`),
   UNIQUE KEY `name` (`name`)
 );