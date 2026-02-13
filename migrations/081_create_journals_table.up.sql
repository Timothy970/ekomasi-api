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
   KEY `idx_journal_entries_entry_date` (`entry_date`)
);


