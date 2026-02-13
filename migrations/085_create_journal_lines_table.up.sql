CREATE TABLE IF NOT EXISTS `journal_entry_lines` (
   `line_id` char(36) NOT NULL,
   `entry_id` char(36) NOT NULL,
   `account_id` char(36) NOT NULL,
   `debit` decimal(15,2) NOT NULL DEFAULT '0.00',
   `credit` decimal(15,2) NOT NULL DEFAULT '0.00',
   `line_description` varchar(255) DEFAULT NULL,
   PRIMARY KEY (`line_id`),
   KEY `idx_journal_entry_lines_entry_id` (`entry_id`),
   KEY `idx_journal_entry_lines_account_id` (`account_id`),
   CONSTRAINT `journal_entry_lines_ibfk_1` FOREIGN KEY (`entry_id`) REFERENCES `journal_entries` (`entry_id`) ON DELETE CASCADE,
   CONSTRAINT `journal_entry_lines_ibfk_2` FOREIGN KEY (`account_id`) REFERENCES `chart_of_accounts` (`account_id`) ON DELETE CASCADE
);