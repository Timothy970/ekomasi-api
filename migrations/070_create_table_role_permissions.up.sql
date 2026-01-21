CREATE TABLE IF NOT EXISTS role_permissions (
    role_permission_id char(36) PRIMARY KEY,
    role_id char(36),
    permission_id char(36),
    UNIQUE (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(role_id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions_master(permission_master_id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ALTER TABLE role_permissions DROP FOREIGN KEY role_permissions_ibfk_2;

-- ALTER TABLE role_permissions ADD FOREIGN KEY (permission_id) REFERENCES permissions_master(permission_master_id) ON DELETE CASCADE;

