ALTER TABLE categories
ADD COLUMN image TEXT DEFAULT NULL AFTER parent_category_id;
