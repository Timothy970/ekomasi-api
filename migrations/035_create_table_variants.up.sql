CREATE TABLE IF NOT EXISTS variants (
   `variant_id` char(36) NOT NULL,
   `variant_type` varchar(100) NOT NULL,
   `name` varchar(100) NOT NULL,
   `hex_code` varchar(7) DEFAULT NULL,
   PRIMARY KEY (`variant_id`),
   UNIQUE KEY `unique_variant` (`variant_type`,`name`));