CREATE TABLE IF NOT EXISTS  `categories` (
   `category_id` char(36) NOT NULL,
   `name` varchar(100) NOT NULL,
   `parent_category_id` char(36) DEFAULT NULL,
   `description` longtext DEFAULT NULL,
   PRIMARY KEY (`category_id`),
   UNIQUE KEY `name` (`name`)
 );