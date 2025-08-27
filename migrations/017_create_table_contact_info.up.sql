CREATE TABLE IF NOT EXISTS  `contact_info` (
   `id` int NOT NULL AUTO_INCREMENT,
   `copyright_text` varchar(255) NOT NULL,
   `company_address` text NOT NULL,
   `contact_email` varchar(100) NOT NULL,
   `phone_number` varchar(30) NOT NULL,
   `last_updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
   PRIMARY KEY (`id`)
 );