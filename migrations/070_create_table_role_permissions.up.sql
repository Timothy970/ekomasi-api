CREATE TABLE IF NOT EXISTS role_permissions (
    role_permission_id char(36) PRIMARY KEY,
    role_id char(36),
    permission_id char(36),
    category VARCHAR(100) NOT NULL DEFAULT 'Users',
    UNIQUE (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(role_id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(permission_id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);