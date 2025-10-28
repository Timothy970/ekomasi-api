-- -- Step 1: null-out invalid JSON values (run these first)
-- ALTER TABLE `blogs`
--     MODIFY COLUMN `content` LONGTEXT DEFAULT NULL;
-- UPDATE `blogs`
-- SET `content` = NULL
-- WHERE JSON_VALID(`content`) = 0 OR JSON_VALID(`content`) IS NULL;

-- UPDATE `blogs`
-- SET `author` = NULL
-- WHERE JSON_VALID(`author`) = 0 OR JSON_VALID(`author`) IS NULL;

-- -- Step 2: alter table (run after the updates)
-- ALTER TABLE `blogs`
--     ADD COLUMN `tags` LONGTEXT DEFAULT NULL AFTER `is_published`,
--     ADD COLUMN `description` TEXT DEFAULT NULL AFTER `tags`,
--     ADD COLUMN `read_time` INT(11) DEFAULT NULL AFTER `description`,
--     ADD COLUMN `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
--     ADD COLUMN `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
--     MODIFY COLUMN `content` JSON DEFAULT NULL,
--     MODIFY COLUMN `author` JSON DEFAULT NULL;

-- ALTER TABLE `blogs`
--     MODIFY COLUMN `published_at` TIMESTAMP NULL DEFAULT NULL;
-- ALTER TABLE `blogs`
--     ADD COLUMN `status` VARCHAR(50) DEFAULT 'published' AFTER `title`;