CREATE TABLE IF NOT EXISTS  `product_bundles` (
   `bundle_id` char(36) NOT NULL,
   `name` varchar(255) NOT NULL,
   `description` text,
   `bundle_price` decimal(10,2) NOT NULL,
   PRIMARY KEY (`bundle_id`)
 );