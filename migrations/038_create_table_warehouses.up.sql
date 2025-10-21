CREATE TABLE IF NOT EXISTS warehouses (
   `warehouse_id` char(36) NOT NULL,
   `name` varchar(100) NOT NULL,
   `location` varchar(255) NOT NULL,
   `warehouse_details` text,
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`warehouse_id`)
 );