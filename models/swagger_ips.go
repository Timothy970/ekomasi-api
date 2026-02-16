package models

import (
	"fmt"
	"time"
)

// SwaggerIP represents an allowed IP address for Swagger access
type SwaggerIP struct {
	ID        int       `json:"id"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
}

// AddAllowedIP adds a new IP address to the allowed list
func AddAllowedIP(db DBExecutor, ip string) error {
	_, err := db.Exec("INSERT INTO swagger_allowed_ips (ip_address) VALUES (?)", ip)
	if err != nil {
		return fmt.Errorf("failed to add allowed IP: %w", err)
	}
	return nil
}

// GetAllowedIPs retrieves all allowed IP addresses
func GetAllowedIPs(db DBExecutor) ([]string, error) {
	rows, err := db.Query("SELECT ip_address FROM swagger_allowed_ips")
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed IPs: %w", err)
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("failed to scan IP: %w", err)
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

// DeleteAllowedIP removes an IP address from the allowed list
func DeleteAllowedIP(db DBExecutor, ip string) error {
	_, err := db.Exec("DELETE FROM swagger_allowed_ips WHERE ip_address = ?", ip)
	if err != nil {
		return fmt.Errorf("failed to delete allowed IP: %w", err)
	}
	return nil
}

// IsIPAllowed checks if a given IP address is in the allowed list
func IsIPAllowed(db DBExecutor, ip string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM swagger_allowed_ips WHERE ip_address = ?", ip).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if IP is allowed: %w", err)
	}
	return count > 0, nil
}
