package models

import (
	"context"
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// GetTenantGatewayConfig retrieves gateway configuration for a specific tenant and gateway name
func GetTenantGatewayConfig(db DBExecutor, tenantID, gatewayName string) (*dtos.TenantPaymentGatewayConfig, error) {
	query := `
		SELECT id, tenant_id, gateway_name, is_enabled, public_key, secret_key, encryption_key, webhook_secret, merchant_id, currency, mode, created_at, updated_at
		FROM tenant_payment_gateways
		WHERE tenant_id = ? AND gateway_name = ?
	`
	var cfg dtos.TenantPaymentGatewayConfig
	var isEnabled int
	err := db.QueryRowContext(context.Background(), query, tenantID, gatewayName).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.GatewayName, &isEnabled, &cfg.PublicKey, &cfg.SecretKey,
		&cfg.EncryptionKey, &cfg.WebhookSecret, &cfg.MerchantID, &cfg.Currency, &cfg.Mode,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("payment gateway configuration not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment gateway config: %w", err)
	}
	cfg.IsEnabled = isEnabled == 1
	return &cfg, nil
}

// GetAllTenantGatewayConfigs retrieves all payment gateway configurations for a tenant
func GetAllTenantGatewayConfigs(db DBExecutor, tenantID string) ([]dtos.TenantPaymentGatewayConfig, error) {
	query := `
		SELECT id, tenant_id, gateway_name, is_enabled, public_key, secret_key, encryption_key, webhook_secret, merchant_id, currency, mode, created_at, updated_at
		FROM tenant_payment_gateways
		WHERE tenant_id = ?
		ORDER BY gateway_name ASC
	`
	rows, err := db.QueryContext(context.Background(), query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenant gateway configs: %w", err)
	}
	defer rows.Close()

	var configs []dtos.TenantPaymentGatewayConfig
	for rows.Next() {
		var cfg dtos.TenantPaymentGatewayConfig
		var isEnabled int
		if err := rows.Scan(
			&cfg.ID, &cfg.TenantID, &cfg.GatewayName, &isEnabled, &cfg.PublicKey, &cfg.SecretKey,
			&cfg.EncryptionKey, &cfg.WebhookSecret, &cfg.MerchantID, &cfg.Currency, &cfg.Mode,
			&cfg.CreatedAt, &cfg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan gateway config row: %w", err)
		}
		cfg.IsEnabled = isEnabled == 1
		configs = append(configs, cfg)
	}
	return configs, nil
}

// SaveTenantGatewayConfig inserts or updates a tenant payment gateway configuration
func SaveTenantGatewayConfig(db DBExecutor, cfg *dtos.TenantPaymentGatewayConfig) error {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	isEnabledInt := 0
	if cfg.IsEnabled {
		isEnabledInt = 1
	}

	query := `
		INSERT INTO tenant_payment_gateways (id, tenant_id, gateway_name, is_enabled, public_key, secret_key, encryption_key, webhook_secret, merchant_id, currency, mode)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			is_enabled = VALUES(is_enabled),
			public_key = VALUES(public_key),
			secret_key = VALUES(secret_key),
			encryption_key = VALUES(encryption_key),
			webhook_secret = VALUES(webhook_secret),
			merchant_id = VALUES(merchant_id),
			currency = VALUES(currency),
			mode = VALUES(mode),
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.ExecContext(context.Background(), query,
		cfg.ID, cfg.TenantID, cfg.GatewayName, isEnabledInt, cfg.PublicKey, cfg.SecretKey,
		cfg.EncryptionKey, cfg.WebhookSecret, cfg.MerchantID, cfg.Currency, cfg.Mode,
	)
	if err != nil {
		return fmt.Errorf("failed to save payment gateway config: %w", err)
	}
	return nil
}
