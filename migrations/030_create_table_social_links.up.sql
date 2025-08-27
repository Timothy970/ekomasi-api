CREATE TABLE IF NOT EXISTS  `social_links` (
   `id` int NOT NULL AUTO_INCREMENT,
   `platform` varchar(50) NOT NULL,
   `url` varchar(255) NOT NULL,
   `icon_class` varchar(50) NOT NULL,
   `display_order` int NOT NULL DEFAULT '0',
   PRIMARY KEY (`id`)
 );