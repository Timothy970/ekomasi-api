CREATE TABLE IF NOT EXISTS `product_bundles` (
   `bundle_id` CHAR(36) NOT NULL,
   `name` VARCHAR(255) NOT NULL,
   `description` TEXT,
   `bundle_price` DECIMAL(10,2) NOT NULL,
   `bundle_image` VARCHAR(255) NOT NULL,
   `compare_at_price` DECIMAL(10,2) DEFAULT NULL,
   `keep_selling_when_out_of_stock` BOOLEAN NOT NULL DEFAULT FALSE,
   `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`bundle_id`)
);
