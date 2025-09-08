CREATE TABLE IF NOT EXISTS variants (
   `variant_id` char(36) NOT NULL,
   `variant_type` enum('color','size','material','brand','gender','age_group','availability','condition','pattern','style','season','fit','capacity','length','width','sales_promotion','feature') NOT NULL,
   `name` varchar(100) NOT NULL,
   `hex_code` varchar(7) DEFAULT NULL,
   PRIMARY KEY (`variant_id`),
   UNIQUE KEY `unique_variant` (`variant_type`,`name`));