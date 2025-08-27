CREATE TABLE IF NOT EXISTS  `menu_links` (
   `id` int NOT NULL AUTO_INCREMENT,
   `title` varchar(100) NOT NULL,
   `url` varchar(255) NOT NULL,
   `display_order` int NOT NULL DEFAULT '0',
   PRIMARY KEY (`id`)
 );