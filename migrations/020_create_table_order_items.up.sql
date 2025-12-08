CREATE TABLE IF NOT EXISTS  `order_items` (
   `order_item_id` char(36) NOT NULL,
   `order_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `variant_id` char(36) NOT NULL,
   `quantity` int NOT NULL,
   `unit_price` decimal(10,2) NOT NULL,
   PRIMARY KEY (`order_item_id`),
   KEY `product_id` (`product_id`),
   KEY `variant_id` (`variant_id`),
   KEY `idx_order_items_order_id` (`order_id`),
   CONSTRAINT `order_items_ibfk_1` FOREIGN KEY (`order_id`) REFERENCES `orders` (`order_id`) ON DELETE CASCADE,
   CONSTRAINT `order_items_ibfk_2` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE CASCADE
 );