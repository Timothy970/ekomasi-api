CREATE TABLE IF NOT EXISTS  `promotions` (
   `promotion_id` char(36) NOT NULL,
   `name` varchar(255) NOT NULL,
   `promotion_type_id` int NOT NULL,
   `start_date` timestamp NOT NULL,
   `end_date` timestamp NOT NULL,
   `is_active` tinyint(1) NOT NULL DEFAULT '1',
   PRIMARY KEY (`promotion_id`),
   KEY `idx_promotions_is_active` (`is_active`)
 );