CREATE TABLE IF NOT EXISTS  `blogs` (
   `blog_id` char(36) NOT NULL,
   `title` varchar(255) NOT NULL,
   `content` text NOT NULL,
   `author_id` char(36) DEFAULT NULL,
   `author` varchar(100) DEFAULT NULL,
   `image_url` varchar(255) DEFAULT NULL,
   `published_at` timestamp NOT NULL,
   `is_published` tinyint(1) NOT NULL DEFAULT '1',
   PRIMARY KEY (`blog_id`),
   KEY `author_id` (`author_id`),
   KEY `idx_blogs_published_at` (`published_at`),
   KEY `idx_blogs_is_published` (`is_published`),
   CONSTRAINT `blogs_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`user_id`) ON DELETE SET NULL
 );