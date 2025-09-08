CREATE TABLE IF NOT EXISTS warehouses (
   `warehouse_id` char(36) NOT NULL,
   `name` varchar(100) NOT NULL,
   `location` varchar(255) NOT NULL,
   `warehouse_details` text,
   PRIMARY KEY (`warehouse_id`)
 );