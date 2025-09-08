CREATE TABLE IF NOT EXISTS purchase_order_items (
   `po_item_id` char(36) NOT NULL,
   `po_id` char(36) NOT NULL,
   `product_id` char(36) NOT NULL,
   `variant_id` char(36) NOT NULL,
   `quantity` int NOT NULL,
   `unit_cost` decimal(10,2) NOT NULL,
   PRIMARY KEY (`po_item_id`),
   KEY `product_id` (`product_id`),
   KEY `variant_id` (`variant_id`),
   KEY `idx_purchase_order_items_po_id` (`po_id`),
   CONSTRAINT `purchase_order_items_ibfk_1` FOREIGN KEY (`po_id`) REFERENCES `purchase_orders` (`po_id`) ON DELETE CASCADE,
   CONSTRAINT `purchase_order_items_ibfk_2` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE RESTRICT
 );