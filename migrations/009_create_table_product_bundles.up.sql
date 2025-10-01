CREATE TABLE IF NOT EXISTS  `product_bundles` (
   `bundle_id` char(36) NOT NULL,
   `name` varchar(255) NOT NULL,
   `description` text,
   `bundle_price` decimal(10,2) NOT NULL,
   `bundle_image` varchar(255) NOT NULL,
   `category_id` char(36) NOT NULL,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`bundle_id`)
   FOREIGN KEY (`category_id`) REFERENCES `categories`(`category_id`) ON DELETE CASCADE
 );