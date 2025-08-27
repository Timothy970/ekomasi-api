CREATE TABLE IF NOT EXISTS  `logs` (
   `log_id` char(36) NOT NULL,
   `level` varchar(50) NOT NULL,
   `message` text NOT NULL,
   `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `user_id` char(36) DEFAULT NULL,
   `metadata` json DEFAULT NULL,
   PRIMARY KEY (`log_id`),
   KEY `idx_logs_timestamp` (`timestamp`),
   KEY `idx_logs_user_id` (`user_id`),
   KEY `idx_logs_level` (`level`)
 );