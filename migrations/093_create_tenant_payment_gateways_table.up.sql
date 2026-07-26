CREATE TABLE IF NOT EXISTS tenant_payment_gateways (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    gateway_name VARCHAR(50) NOT NULL, -- 'paystack', 'flutterwave', 'stripe'
    is_enabled TINYINT(1) DEFAULT 0,
    public_key VARCHAR(255) DEFAULT '',
    secret_key VARCHAR(255) DEFAULT '',
    encryption_key VARCHAR(255) DEFAULT '',
    webhook_secret VARCHAR(255) DEFAULT '',
    merchant_id VARCHAR(255) DEFAULT '',
    currency VARCHAR(10) DEFAULT 'KES',
    mode VARCHAR(20) DEFAULT 'test', -- 'test' or 'live'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_tenant_gateway (tenant_id, gateway_name),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
