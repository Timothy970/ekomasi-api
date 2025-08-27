CREATE TABLE IF NOT EXISTS  `delivery_rates` (
   `id` bigint unsigned NOT NULL AUTO_INCREMENT,
   `location` varchar(255) NOT NULL,
   `charge` decimal(10,2) NOT NULL,
   PRIMARY KEY (`id`),
   UNIQUE KEY `id` (`id`)
 );