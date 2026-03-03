CREATE TABLE IF NOT EXISTS  `blogs` (
   `blog_id` char(36) NOT NULL,
   `title` varchar(255) NOT NULL,
   `content` json NOT NULL,
   `author_id` char(36) DEFAULT NULL,
   `author` json DEFAULT NULL,
   `banner_image_url` varchar(255) DEFAULT NULL,
   `status` varchar(50) DEFAULT 'published',
   `published_at` timestamp NOT NULL,
   `is_published` tinyint(1) NOT NULL DEFAULT '1',
   `tags` json DEFAULT NULL,
   `description` LONGTEXT DEFAULT NULL,
   `read_time` int(11) DEFAULT NULL,
   PRIMARY KEY (`blog_id`),
   KEY `author_id` (`author_id`),
   KEY `idx_blogs_published_at` (`published_at`),
   KEY `idx_blogs_is_published` (`is_published`),
   CONSTRAINT `blogs_ibfk_1` FOREIGN KEY (`author_id`) REFERENCES `users` (`user_id`) ON DELETE SET NULL
 );