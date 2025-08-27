CREATE TABLE IF NOT EXISTS  `analytics_sales` (
   `sales_id` char(36) NOT NULL,
   `date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `product_id` char(36) NOT NULL,
   `quantity_sold` int NOT NULL,
   `total_revenue` decimal(10,2) NOT NULL,
   PRIMARY KEY (`sales_id`),
   KEY `idx_analytics_sales_date` (`date`),
   KEY `idx_analytics_sales_product_id` (`product_id`),
   KEY `idx_analytics_sales_date_product` (`date`,`product_id`),
   CONSTRAINT `analytics_sales_ibfk_1` FOREIGN KEY (`product_id`) REFERENCES `products` (`product_id`) ON DELETE RESTRICT
 );