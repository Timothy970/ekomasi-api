CREATE TABLE IF NOT EXISTS suppliers (
   `supplier_id` char(36) NOT NULL,
   `name` varchar(100) NOT NULL,
   `contact_email` varchar(255) DEFAULT NULL,
   `contact_phone` varchar(20) DEFAULT NULL,
   `extra_details` text,
   PRIMARY KEY (`supplier_id`)
 );