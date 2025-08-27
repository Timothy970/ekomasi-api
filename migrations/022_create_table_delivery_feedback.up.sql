CREATE TABLE IF NOT EXISTS  `delivery_feedback` (
   `delivery_feedback_id` char(36) NOT NULL,
   `delivery_id` char(36) NOT NULL,
   `score` int NOT NULL,
   `details` text,
   PRIMARY KEY (`delivery_feedback_id`),
   KEY `idx_delivery_feedback_delivery_id` (`delivery_id`),
   CONSTRAINT `delivery_feedback_ibfk_1` FOREIGN KEY (`delivery_id`) REFERENCES `deliveries` (`delivery_id`) ON DELETE CASCADE
 );