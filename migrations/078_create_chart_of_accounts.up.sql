CREATE TABLE IF NOT EXISTS `chart_of_accounts` (
   `account_id` char(36) NOT NULL,
   `account_code` varchar(50) NOT NULL,
   `account_name` varchar(100) NOT NULL,
   `account_type` varchar(50) NOT NULL,
   `balance` decimal(10,2) NOT NULL DEFAULT '0.00',
   `statement_type` varchar(50) NOT NULL,
   `description` text,
   `status` varchar(20) NOT NULL,
   `category` varchar(50) NOT NULL,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`account_id`),
   UNIQUE KEY `account_code` (`account_code`),
   KEY `idx_chart_of_accounts_account_code` (`account_code`)
 );