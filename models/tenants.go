package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Tenant struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	Domain              string    `json:"domain"`
	AppDomain           string    `json:"app_domain"`
	AdminDomain         string    `json:"admin_domain"`
	Slogan              string    `json:"slogan"`
	Logo                string    `json:"logo"`
	Color               string    `json:"color"`
	AppLogo             string    `json:"app_logo"`
	AppPrimaryColor     string    `json:"app_primary_color"`
	AppSecondaryColor   string    `json:"app_secondary_color"`
	AppTertiaryColor    string    `json:"app_tertiary_color"`
	AdminLogo           string    `json:"admin_logo"`
	AdminPrimaryColor   string    `json:"admin_primary_color"`
	AdminSecondaryColor string    `json:"admin_secondary_color"`
	AdminTertiaryColor  string    `json:"admin_tertiary_color"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func CreateTenant(
	db DBExecutor,
	name, domain, appDomain, adminDomain, slogan, logo, color,
	appLogo, appPrimaryColor, appSecondaryColor, appTertiaryColor,
	adminLogo, adminPrimaryColor, adminSecondaryColor, adminTertiaryColor string,
) error {
	if appDomain == "" {
		appDomain = domain
	}
	if adminDomain == "" {
		adminDomain = domain
	}
	if appLogo == "" {
		appLogo = logo
	}
	if appPrimaryColor == "" {
		appPrimaryColor = color
	}
	if appSecondaryColor == "" {
		appSecondaryColor = "#E8298A"
	}
	if appTertiaryColor == "" {
		appTertiaryColor = "#EEF2FF"
	}
	if adminLogo == "" {
		adminLogo = logo
	}
	if adminPrimaryColor == "" {
		adminPrimaryColor = "#0f172a"
	}
	if adminSecondaryColor == "" {
		adminSecondaryColor = "#AF52DE"
	}
	if adminTertiaryColor == "" {
		adminTertiaryColor = "#1B202E"
	}

	exists, err := RecordExists(db, "tenants", "domain = ? OR app_domain = ? OR admin_domain = ?", domain, appDomain, adminDomain)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tenant with matching domain already exists")
	}

	query := `
		INSERT INTO tenants (
			name, domain, app_domain, admin_domain, slogan, logo, color,
			app_logo, app_primary_color, app_secondary_color, app_tertiary_color,
			admin_logo, admin_primary_color, admin_secondary_color, admin_tertiary_color
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(
		query,
		name, domain, appDomain, adminDomain, slogan, logo, color,
		appLogo, appPrimaryColor, appSecondaryColor, appTertiaryColor,
		adminLogo, adminPrimaryColor, adminSecondaryColor, adminTertiaryColor,
	)
	return err
}

func UpdateTenant(
	db DBExecutor, id int,
	name, domain, appDomain, adminDomain, slogan, logo, color,
	appLogo, appPrimaryColor, appSecondaryColor, appTertiaryColor,
	adminLogo, adminPrimaryColor, adminSecondaryColor, adminTertiaryColor string,
) error {
	exists, err := RecordExists(db, "tenants", "id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("tenant not found")
	}

	query := `
		UPDATE tenants
		SET name = ?, domain = ?, app_domain = ?, admin_domain = ?, slogan = ?, logo = ?, color = ?,
		    app_logo = ?, app_primary_color = ?, app_secondary_color = ?, app_tertiary_color = ?,
		    admin_logo = ?, admin_primary_color = ?, admin_secondary_color = ?, admin_tertiary_color = ?
		WHERE id = ?
	`
	_, err = db.Exec(
		query,
		name, domain, appDomain, adminDomain, slogan, logo, color,
		appLogo, appPrimaryColor, appSecondaryColor, appTertiaryColor,
		adminLogo, adminPrimaryColor, adminSecondaryColor, adminTertiaryColor,
		id,
	)
	return err
}

func GetTenantByID(db DBExecutor, id int) (*Tenant, error) {
	query := `
		SELECT id, name, domain, 
		       COALESCE(app_domain, domain) as app_domain, 
		       COALESCE(admin_domain, domain) as admin_domain, 
		       slogan, logo, color, 
		       COALESCE(app_logo, logo) as app_logo, 
		       COALESCE(app_primary_color, color, '#4f46e5') as app_primary_color, 
		       COALESCE(app_secondary_color, '#E8298A') as app_secondary_color, 
		       COALESCE(app_tertiary_color, '#EEF2FF') as app_tertiary_color, 
		       COALESCE(admin_logo, logo) as admin_logo, 
		       COALESCE(admin_primary_color, '#0f172a') as admin_primary_color, 
		       COALESCE(admin_secondary_color, '#AF52DE') as admin_secondary_color, 
		       COALESCE(admin_tertiary_color, '#1B202E') as admin_tertiary_color, 
		       created_at, updated_at
		FROM tenants
		WHERE id = ?
	`
	var t Tenant
	err := db.QueryRow(query, id).Scan(
		&t.ID, &t.Name, &t.Domain, &t.AppDomain, &t.AdminDomain, &t.Slogan, &t.Logo, &t.Color,
		&t.AppLogo, &t.AppPrimaryColor, &t.AppSecondaryColor, &t.AppTertiaryColor,
		&t.AdminLogo, &t.AdminPrimaryColor, &t.AdminSecondaryColor, &t.AdminTertiaryColor,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}
	return &t, nil
}

func GetTenantByDomain(db DBExecutor, domain string) (*Tenant, error) {
	cleanDomain := domain
	if idx := strings.Index(domain, ":"); idx != -1 {
		cleanDomain = domain[:idx]
	}

	query := `
		SELECT id, name, domain, 
		       COALESCE(app_domain, domain) as app_domain, 
		       COALESCE(admin_domain, domain) as admin_domain, 
		       slogan, logo, color, 
		       COALESCE(app_logo, logo) as app_logo, 
		       COALESCE(app_primary_color, color, '#4f46e5') as app_primary_color, 
		       COALESCE(app_secondary_color, '#E8298A') as app_secondary_color, 
		       COALESCE(app_tertiary_color, '#EEF2FF') as app_tertiary_color, 
		       COALESCE(admin_logo, logo) as admin_logo, 
		       COALESCE(admin_primary_color, '#0f172a') as admin_primary_color, 
		       COALESCE(admin_secondary_color, '#AF52DE') as admin_secondary_color, 
		       COALESCE(admin_tertiary_color, '#1B202E') as admin_tertiary_color, 
		       created_at, updated_at
		FROM tenants
		WHERE domain = ? OR app_domain = ? OR admin_domain = ?
		   OR domain LIKE ? OR app_domain LIKE ? OR admin_domain LIKE ?
		ORDER BY id ASC LIMIT 1
	`
	likePattern := cleanDomain + "%"
	var t Tenant
	err := db.QueryRow(query, domain, domain, domain, likePattern, likePattern, likePattern).Scan(
		&t.ID, &t.Name, &t.Domain, &t.AppDomain, &t.AdminDomain, &t.Slogan, &t.Logo, &t.Color,
		&t.AppLogo, &t.AppPrimaryColor, &t.AppSecondaryColor, &t.AppTertiaryColor,
		&t.AdminLogo, &t.AdminPrimaryColor, &t.AdminSecondaryColor, &t.AdminTertiaryColor,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return GetTenantByID(db, 1)
		}
		return nil, err
	}
	return &t, nil
}

func GetAllTenants(db DBExecutor) ([]Tenant, error) {
	query := `
		SELECT id, name, domain, 
		       COALESCE(app_domain, domain) as app_domain, 
		       COALESCE(admin_domain, domain) as admin_domain, 
		       slogan, logo, color, 
		       COALESCE(app_logo, logo) as app_logo, 
		       COALESCE(app_primary_color, color, '#4f46e5') as app_primary_color, 
		       COALESCE(app_secondary_color, '#E8298A') as app_secondary_color, 
		       COALESCE(app_tertiary_color, '#EEF2FF') as app_tertiary_color, 
		       COALESCE(admin_logo, logo) as admin_logo, 
		       COALESCE(admin_primary_color, '#0f172a') as admin_primary_color, 
		       COALESCE(admin_secondary_color, '#AF52DE') as admin_secondary_color, 
		       COALESCE(admin_tertiary_color, '#1B202E') as admin_tertiary_color, 
		       created_at, updated_at
		FROM tenants
		ORDER BY id ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Domain, &t.AppDomain, &t.AdminDomain, &t.Slogan, &t.Logo, &t.Color,
			&t.AppLogo, &t.AppPrimaryColor, &t.AppSecondaryColor, &t.AppTertiaryColor,
			&t.AdminLogo, &t.AdminPrimaryColor, &t.AdminSecondaryColor, &t.AdminTertiaryColor,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func DeleteTenant(db DBExecutor, id int) error {
	exists, err := RecordExists(db, "tenants", "id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("tenant not found")
	}

	_, err = db.Exec("DELETE FROM tenants WHERE id = ?", id)
	return err
}
