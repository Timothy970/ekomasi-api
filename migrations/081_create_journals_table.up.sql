-- Journal Entries Header Table
-- Contains the main entry information that can affect multiple accounts
CREATE TABLE IF NOT EXISTS `journal_entries` (
   `entry_id` char(36) NOT NULL,
   `entry_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `description` varchar(255) DEFAULT NULL,
   `reference` varchar(255) DEFAULT NULL,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`entry_id`),
   KEY `idx_journal_entries_entry_date` (`entry_date`),
);

-- Journal Entry Lines Table
-- Contains the individual account debits and credits for each entry
-- Multiple lines per entry to support double-entry bookkeeping
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

