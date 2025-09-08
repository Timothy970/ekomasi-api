CREATE TABLE IF NOT EXIsTS purchase_orders (
   `po_id` char(36) NOT NULL,
   `supplier_id` char(36) DEFAULT NULL,
   `status` varchar(50) NOT NULL DEFAULT 'pending',
   `total_cost` decimal(10,2) NOT NULL,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `approved_at` timestamp NULL DEFAULT NULL,
   PRIMARY KEY (`po_id`),
   KEY `idx_purchase_orders_supplier_id` (`supplier_id`),
   CONSTRAINT `purchase_orders_ibfk_1` FOREIGN KEY (`supplier_id`) REFERENCES `suppliers` (`supplier_id`) ON DELETE RESTRICT
 );