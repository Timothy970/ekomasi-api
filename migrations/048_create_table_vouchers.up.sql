CREATE TABLE IF NOT EXISTS `vouchers` (
  `voucher_id` char(36) NOT NULL,
  `user_id` char(36) DEFAULT NULL,
  `code` varchar(50) NOT NULL,
  `balance` decimal(10,2) DEFAULT '0.00',
  `original_value` decimal(10,2) NOT NULL DEFAULT '0.00',
  `expiry_date` datetime NOT NULL,
  `status` varchar(20) DEFAULT 'inactive',
  `is_redeemed` tinyint(1) DEFAULT '0',
  `notes` longtext,
  `design_id` char(36) DEFAULT NULL,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`voucher_id`),
  UNIQUE KEY `code` (`code`),
  KEY `v_user` (`user_id`),
  CONSTRAINT `v_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE
);