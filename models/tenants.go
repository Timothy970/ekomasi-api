package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Tenant represents a registered storefront tenant in the system.
type Tenant struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Domain    string       `json:"domain"` // canonical slug/reference, kept for display & logs
	Slogan    string       `json:"slogan"`
	Logo      string       `json:"logo"`
	AppLogo   string       `json:"app_logo"`
	AdminLogo string       `json:"admin_logo"`
	URLs      []TenantURL  `json:"urls,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// TenantURL represents a single domain/URL entry linked to a tenant.
type TenantURL struct {
	ID        int       `json:"id"`
	TenantID  int       `json:"tenant_id"`
	URL       string    `json:"url"`
	URLType   string    `json:"url_type"` // "storefront" | "admin"
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const conditionIDEquals = "id = ?"

// ─── Tenant CRUD ────────────────────────────────────────────────────────────

// CreateTenantParams holds all parameters for creating a new tenant.
type CreateTenantParams struct {
	Name          string
	Domain        string
	Slogan        string
	Logo          string
	AppLogo       string
	AdminLogo     string
	StorefrontURL string
	AdminURL      string
}

// CreateTenant inserts a new tenant record. The initial storefront and admin
// domains are registered as primary entries in tenant_urls.
func CreateTenant(db DBExecutor, p CreateTenantParams) error {
	if p.AppLogo == "" {
		p.AppLogo = p.Logo
	}
	if p.AdminLogo == "" {
		p.AdminLogo = p.Logo
	}
	if p.StorefrontURL == "" {
		p.StorefrontURL = p.Domain
	}

	// Check domain uniqueness against tenant_urls
	exists, err := RecordExists(db, "tenant_urls", "url = ?", p.StorefrontURL)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("a tenant with URL %q already exists", p.StorefrontURL)
	}

	query := `
		INSERT INTO tenants (name, domain, slogan, logo, app_logo, admin_logo)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := db.Exec(query, p.Name, p.Domain, p.Slogan, p.Logo, p.AppLogo, p.AdminLogo)
	if err != nil {
		return err
	}

	tenantID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	id := int(tenantID)

	// Register storefront URL as primary
	if err := AddTenantURL(db, id, p.StorefrontURL, "storefront", true); err != nil {
		return err
	}

	// Register admin URL if provided and different
	if p.AdminURL != "" && p.AdminURL != p.StorefrontURL {
		if err := AddTenantURL(db, id, p.AdminURL, "admin", true); err != nil {
			return err
		}
	}

	return nil
}

// UpdateTenant updates mutable tenant fields (name, slogan, logos).
// Domain routing is managed separately via the TenantURL CRUD.
func UpdateTenant(
	db DBExecutor, id int,
	name, slogan, logo, appLogo, adminLogo string,
) error {
	exists, err := RecordExists(db, "tenants", conditionIDEquals, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("tenant not found")
	}

	query := `
		UPDATE tenants
		SET name = ?, slogan = ?, logo = ?, app_logo = ?, admin_logo = ?
		WHERE id = ?
	`
	_, err = db.Exec(query, name, slogan, logo, appLogo, adminLogo, id)
	return err
}

// GetTenantByID retrieves a tenant along with all its registered URLs.
func GetTenantByID(db DBExecutor, id int) (*Tenant, error) {
	query := `
		SELECT id, name, domain,
		       COALESCE(slogan, '') as slogan,
		       COALESCE(logo, '') as logo,
		       COALESCE(app_logo, logo, '') as app_logo,
		       COALESCE(admin_logo, logo, '') as admin_logo,
		       created_at, updated_at
		FROM tenants
		WHERE id = ?
	`
	var t Tenant
	err := db.QueryRow(query, id).Scan(
		&t.ID, &t.Name, &t.Domain, &t.Slogan, &t.Logo,
		&t.AppLogo, &t.AdminLogo, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}

	urls, err := GetTenantURLs(db, t.ID)
	if err != nil {
		return nil, err
	}
	t.URLs = urls

	return &t, nil
}

// GetTenantByDomain resolves the tenant whose tenant_urls table contains an
// exact match for the given domain string (port-stripped or full).
// Falls back to tenant ID=1 if no match found.
func GetTenantByDomain(db DBExecutor, domain string) (*Tenant, error) {
	cleanDomain := domain
	if idx := strings.Index(domain, ":"); idx != -1 {
		cleanDomain = domain[:idx]
	}

	query := `
		SELECT t.id, t.name, t.domain,
		       COALESCE(t.slogan, '') as slogan,
		       COALESCE(t.logo, '') as logo,
		       COALESCE(t.app_logo, t.logo, '') as app_logo,
		       COALESCE(t.admin_logo, t.logo, '') as admin_logo,
		       t.created_at, t.updated_at
		FROM tenants t
		JOIN tenant_urls u ON u.tenant_id = t.id
		WHERE u.url = ? OR u.url = ?
		ORDER BY t.id ASC
		LIMIT 1
	`
	var t Tenant
	err := db.QueryRow(query, domain, cleanDomain).Scan(
		&t.ID, &t.Name, &t.Domain, &t.Slogan, &t.Logo,
		&t.AppLogo, &t.AdminLogo, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return GetTenantByID(db, 1)
		}
		return nil, err
	}

	urls, err := GetTenantURLs(db, t.ID)
	if err != nil {
		return nil, err
	}
	t.URLs = urls

	return &t, nil
}

// GetAllTenants retrieves all tenants along with their registered URLs.
func GetAllTenants(db DBExecutor) ([]Tenant, error) {
	query := `
		SELECT id, name, domain,
		       COALESCE(slogan, '') as slogan,
		       COALESCE(logo, '') as logo,
		       COALESCE(app_logo, logo, '') as app_logo,
		       COALESCE(admin_logo, logo, '') as admin_logo,
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
			&t.ID, &t.Name, &t.Domain, &t.Slogan, &t.Logo,
			&t.AppLogo, &t.AdminLogo, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load URLs for each tenant
	for i := range tenants {
		urls, err := GetTenantURLs(db, tenants[i].ID)
		if err != nil {
			return nil, err
		}
		tenants[i].URLs = urls
	}

	return tenants, nil
}

// DeleteTenant removes a tenant. All associated tenant_urls are cascade-deleted.
func DeleteTenant(db DBExecutor, id int) error {
	exists, err := RecordExists(db, "tenants", conditionIDEquals, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("tenant not found")
	}

	_, err = db.Exec("DELETE FROM tenants WHERE id = ?", id)
	return err
}

// ─── Tenant URL CRUD ─────────────────────────────────────────────────────────

// GetTenantURLs retrieves all URLs registered for a tenant.
func GetTenantURLs(db DBExecutor, tenantID int) ([]TenantURL, error) {
	query := `
		SELECT id, tenant_id, url, url_type, is_primary, created_at, updated_at
		FROM tenant_urls
		WHERE tenant_id = ?
		ORDER BY is_primary DESC, url_type ASC, id ASC
	`
	rows, err := db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []TenantURL
	for rows.Next() {
		var u TenantURL
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.URL, &u.URLType, &u.IsPrimary,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, rows.Err()
}

// AddTenantURL registers a new URL for a tenant.
// If isPrimary is true, it demotes any existing primary of the same url_type.
func AddTenantURL(db DBExecutor, tenantID int, url, urlType string, isPrimary bool) error {
	if url == "" {
		return errors.New("url cannot be empty")
	}
	if urlType != "storefront" && urlType != "admin" {
		return errors.New("url_type must be 'storefront' or 'admin'")
	}

	// Check uniqueness globally across all tenants
	exists, err := RecordExists(db, "tenant_urls", "url = ?", url)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("URL %q is already registered to a tenant", url)
	}

	// Demote existing primary for same type if this one is primary
	if isPrimary {
		_, err = db.Exec(
			"UPDATE tenant_urls SET is_primary = 0 WHERE tenant_id = ? AND url_type = ? AND is_primary = 1",
			tenantID, urlType,
		)
		if err != nil {
			return err
		}
	}

	_, err = db.Exec(
		"INSERT INTO tenant_urls (tenant_id, url, url_type, is_primary) VALUES (?, ?, ?, ?)",
		tenantID, url, urlType, isPrimary,
	)
	return err
}

// DeleteTenantURL removes a URL entry by its ID.
func DeleteTenantURL(db DBExecutor, urlID int) error {
	exists, err := RecordExists(db, "tenant_urls", conditionIDEquals, urlID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("tenant URL not found")
	}
	_, err = db.Exec("DELETE FROM tenant_urls WHERE id = ?", urlID)
	return err
}

// SetTenantURLPrimary marks a given URL as primary for its url_type, demoting others.
func SetTenantURLPrimary(db DBExecutor, urlID, tenantID int, urlType string) error {
	_, err := db.Exec(
		"UPDATE tenant_urls SET is_primary = 0 WHERE tenant_id = ? AND url_type = ?",
		tenantID, urlType,
	)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		"UPDATE tenant_urls SET is_primary = 1 WHERE id = ? AND tenant_id = ?",
		urlID, tenantID,
	)
	return err
}
