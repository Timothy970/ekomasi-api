CREATE TABLE IF NOT EXISTS  `coupons` (
   `coupon_id` varchar(36) NOT NULL,
   `code` varchar(50) NOT NULL,
   `discount_pct` int NOT NULL,
   `expires_at` timestamp NULL DEFAULT NULL,
   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (`coupon_id`),
   UNIQUE KEY `code` (`code`),
   CONSTRAINT `coupons_chk_1` CHECK ((`discount_pct` between 1 and 100))
 );