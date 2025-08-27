ALTER TABLE menu_links
DROP CONSTRAINT fk_menu_links_parent,
DROP COLUMN parent_id;
