CREATE TABLE IF NOT EXISTS subscribers (
   `subscriber_id` char(36) NOT NULL,
   `email` varchar(100) NOT NULL,
   `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (`subscriber_id`)
 );