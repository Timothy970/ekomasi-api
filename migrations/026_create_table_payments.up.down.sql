CREATE TABLE IF NOT EXISTS  `payments` (
   `payment_id` char(36) NOT NULL,
   `order_id` char(36) DEFAULT NULL,
   `amount` decimal(10,2) NOT NULL,
   `voucher_id` char(36) DEFAULT NULL,
   `status` varchar(50) NOT NULL DEFAULT 'pending',
   `payment_method` varchar(50) NOT NULL DEFAULT 'mpesa',
   `transaction_id` varchar(100) DEFAULT NULL,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (`payment_id`),
   UNIQUE KEY `transaction_id` (`transaction_id`),
   KEY `voucher_id` (`voucher_id`),
   KEY `idx_payments_order_id` (`order_id`),
   KEY `idx_payments_transaction_id` (`transaction_id`),
   CONSTRAINT `payments_ibfk_1` FOREIGN KEY (`order_id`) REFERENCES `orders` (`order_id`) ON DELETE RESTRICT
 );