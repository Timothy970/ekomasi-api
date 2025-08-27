CREATE TABLE IF NOT EXISTS users (
    `user_id` char(50) NOT NULL,
    `email` varchar(255) DEFAULT NULL,
    `phone_number` varchar(20) DEFAULT NULL,
    `first_name` varchar(100) DEFAULT NULL,
    `last_name` varchar(100) DEFAULT NULL,
    `role` varchar(100) NOT NULL DEFAULT 'customer',
    `status` varchar(20) DEFAULT 'active',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `last_login` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`user_id`),
    UNIQUE KEY `email` (`email`),
    UNIQUE KEY `phone_number` (`phone_number`),
    KEY `idx_users_email` (`email`),
    KEY `idx_users_phone_number` (`phone_number`)
 );