CREATE TABLE IF NOT EXISTS mpesa_paybills (
    `id` CHAR(36) NOT NULL,
    `paybill_number` VARCHAR(20) NOT NULL UNIQUE,
    `account_reference` VARCHAR(100) DEFAULT NULL,
    `consumer_key_secret_name` VARCHAR(255) DEFAULT NULL, 
    `consumer_secret_secret_name` VARCHAR(255) DEFAULT NULL, 
    `passkey_secret_name` VARCHAR(255) DEFAULT NULL,      
    `status` ENUM('active', 'inactive') NOT NULL DEFAULT 'active',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY idx_paybill_number (paybill_number),
    KEY idx_status (status)
);
