CREATE TABLE IF NOT EXISTS  `stk_push_responses` (
   `id` bigint NOT NULL AUTO_INCREMENT,
   `order_id` varchar(100) NOT NULL,
   `delivery_id` varchar(100) DEFAULT NULL,
   `amount` decimal(10,2) NOT NULL,
   `checkout_request_id` varchar(100) DEFAULT NULL,
   `merchant_request_id` varchar(100) DEFAULT NULL,
   `status` varchar(50) DEFAULT NULL,
   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (`id`)
 );