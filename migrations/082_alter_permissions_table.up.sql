ALTER TABLE permissions
    DROP COLUMN name,
    DROP COLUMN category,
    DROP COLUMN permission_key,
    ADD COLUMN category VARCHAR(100) UNIQUE NOT NULL AFTER permission_id,
    ADD COLUMN description TEXT DEFAULT NULL AFTER category,
    ADD COLUMN permission_key VARCHAR(100) UNIQUE NOT NULL AFTER description;