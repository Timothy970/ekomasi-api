CREATE TABLE IF NOT EXISTS `logs` (
   `log_id` CHAR(36) NOT NULL,
   `level` VARCHAR(50) NOT NULL,
   `message` TEXT NOT NULL,
   `timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `user_id` CHAR(36) DEFAULT NULL,
   `metadata` JSON DEFAULT NULL,
   PRIMARY KEY (`log_id`),

   KEY `idx_logs_timestamp` (`timestamp`),
   KEY `idx_logs_user_id` (`user_id`),
   KEY `idx_logs_level` (`level`),

   KEY `idx_logs_userid_timestamp` (`user_id`, `timestamp` DESC),
   KEY `idx_logs2_timestamp` (`timestamp` DESC),
   KEY `idx_logs_module` (`module`),
   KEY `idx_logs_role` (`role`)
);


