CREATE TABLE IF NOT EXISTS  `product_bundles` (
   `bundle_id` char(36) NOT NULL,
   `name` varchar(255) NOT NULL,
   `description` text,
   `bundle_price` decimal(10,2) NOT NULL,
   `bundle_image` varchar(255) NOT NULL,
   `category_id` char(36) NOT NULL,
   `compare_at_price` decimal(10,2) DEFAULT NULL,
   `keep_selling_when_out_of_stock` boolean NOT NULL DEFAULT FALSE,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`bundle_id`),
   CONSTRAINT `fk_product_bundles_categories` FOREIGN KEY (`category_id`) REFERENCES `categories`(`category_id`) ON DELETE CASCADE
 );