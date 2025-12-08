CREATE TABLE IF NOT EXISTS  `deliveries` (
   `delivery_id` char(36) NOT NULL,
   `order_id` char(36) NOT NULL,
   `delivery_charge` varchar(255) NOT NULL,
   `status` varchar(100) NOT NULL DEFAULT 'pending',
   `delivery_address` varchar(500) DEFAULT NULL,
   `courier_details` text,
   `delivered_at` datetime DEFAULT NULL,
   PRIMARY KEY (`delivery_id`),
   KEY `idx_deliveries_order_id` (`order_id`),
   CONSTRAINT `fk_deliveries_order_id` FOREIGN KEY (`order_id`) REFERENCES `orders` (`order_id`) ON DELETE CASCADE
 );

