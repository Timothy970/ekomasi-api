package utils

import (
	"adenzo_backend/middleware"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

var tagToMessage = map[string]string{
	"required": "is required and cannot be empty",
	"gt":       "must be greater than zero",
	"gte":      "must be greater than or equal to zero",
	"email":    "must be a valid email address",
	"min":      "does not meet the minimum length",
	"max":      "exceeds the maximum allowed length",
	"len":      "must have a fixed length",
}

// ValidateStruct returns map of field errors with friendly messages
func ValidateStruct(data any) map[string]string {
	errs := make(map[string]string)

	err := validate.Struct(data)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			fieldName := e.Field()
			message, ok := tagToMessage[e.Tag()]
			if !ok {
				message = fmt.Sprintf("failed validation on '%s'", e.Tag())
			}
			errs[fieldName] = fmt.Sprintf("%s %s", fieldName, message)
		}
	}

	return errs
}

// ValidateStructAndRespond validates, responds with 400 if errors, returns (isValid)
var ValidateStructAndRespond = func(
	data any,
	w http.ResponseWriter,
	r *http.Request,
	requestSummary string,
	start time.Time,
	module string,
) bool {
	validationErrors := ValidateStruct(data)

	if len(validationErrors) > 0 {
		RespondWithJSON(w, SuccessJSONResponseOptions{
			CollectiveInfo: CollectiveInfo{
				Module:      module,
				Description: "Validation failed for the provided input",
				Code:        http.StatusBadRequest,
			},
			Payload:   validationErrors,
			Message:   "Validation failed",
			TimeTaken: time.Since(start),
			Function:  GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return false
	}

	return true
}

var RequireAdmin = func(
	r *http.Request,
	w http.ResponseWriter,
	start time.Time,
	requestSummary string,
	module string,
) (middleware.AuthenticatedUser, bool) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		RespondWithError(w, ErrorJSONResponseOptions{
			CollectiveInfo: CollectiveInfo{
				Module:      module,
				Description: "Authentication failed",
				Code:        http.StatusForbidden,
			},
			Message:   "User is not validated",
			TimeTaken: time.Since(start),
			Function:  GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return middleware.AuthenticatedUser{}, false
	}

	if user.Role != "admin" {
		RespondWithError(w, ErrorJSONResponseOptions{
			CollectiveInfo: CollectiveInfo{
				Module:      module,
				Description: "Authorization failed",
				Code:        http.StatusForbidden,
			},
			Message:   "This action requires admin role",
			TimeTaken: time.Since(start),
			Function:  GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return middleware.AuthenticatedUser{}, false
	}

	return user, true
}

// RequirePermissions ensures that the authenticated user has the specified permissions.
// It checks the user's role, retrieves permissions (from cache or DB), and validates access.
func RequirePermissions(db *sql.DB, requiredPerms []string, module string) func(
	r *http.Request,
	w http.ResponseWriter,
	start time.Time,
	requestSummary string,
) (middleware.AuthenticatedUser, bool) {

	return func(r *http.Request, w http.ResponseWriter, start time.Time, requestSummary string) (middleware.AuthenticatedUser, bool) {
		user, ok := middleware.UserFromContext(r.Context())
		if !ok {
			sendError(w, http.StatusForbidden, "User not validated", start, r, requestSummary, module)
			return middleware.AuthenticatedUser{}, false
		}

		roleID, err := fetchUserRole(db, user.ID)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to fetch user role: "+err.Error(), start, r, requestSummary, module)
			return middleware.AuthenticatedUser{}, false
		}

		rolePerms, err := getRolePermissions(db, roleID)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to fetch role permissions: "+err.Error(), start, r, requestSummary, module)
			return middleware.AuthenticatedUser{}, false
		}
		hasAccess, missingPerms := hasAllRequiredPermissions(rolePerms, requiredPerms)
		if !hasAccess {
			errorMsg := fmt.Sprintf("Missing required permissions: %s", strings.Join(missingPerms, ", "))
			sendError(w, http.StatusForbidden, errorMsg, start, r, requestSummary, module)
			return middleware.AuthenticatedUser{}, false
		}

		return user, true
	}
}

// fetchUserRole retrieves the user's role_id from the database.
func fetchUserRole(db *sql.DB, userID string) (string, error) {
	var roleID string
	err := db.QueryRow(`SELECT role_id FROM users WHERE id = ?`, userID).Scan(&roleID)
	return roleID, err
}

// getRolePermissions returns all permissions associated with a role, using cache if available.
func getRolePermissions(db *sql.DB, roleID string) ([]string, error) {
	cacheKey := "role_permissions:" + roleID
	var rolePerms []string

	// Try to get from cache
	if err := GetCache(cacheKey, &rolePerms); err == nil {
		return rolePerms, nil
	}

	// Cache miss — fetch from DB
	rows, err := db.Query(`
		SELECT p.name
		FROM role_permissions rp
		JOIN permissions p ON p.permission_id = rp.permission_id
		WHERE rp.role_id = ?`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var permName string
		if err := rows.Scan(&permName); err == nil {
			rolePerms = append(rolePerms, permName)
		}
	}

	// Cache the result (non-blocking)
	_ = SetCache(cacheKey, rolePerms)
	return rolePerms, nil
}

// hasAllRequiredPermissions verifies that the role includes all required permissions and returns missing ones.
func hasAllRequiredPermissions(rolePerms, requiredPerms []string) (bool, []string) {
	rolePermsMap := make(map[string]bool, len(rolePerms))
	for _, perm := range rolePerms {
		rolePermsMap[strings.ToLower(perm)] = true
	}

	var missingPerms []string
	for _, required := range requiredPerms {
		if !rolePermsMap[strings.ToLower(required)] {
			missingPerms = append(missingPerms, required)
		}
	}

	return len(missingPerms) == 0, missingPerms
}

// sendError standardizes error responses for this middleware.
func sendError(w http.ResponseWriter, code int, message string, start time.Time, r *http.Request, summary string, module string) {
	RespondWithError(w, ErrorJSONResponseOptions{
		CollectiveInfo: CollectiveInfo{
			Module:      module,
			Description: "Permission check failed",
			Code:        code,
		},
		Message:   message,
		TimeTaken: time.Since(start),
		Function:  GetCurrentFuncName(),
		Request:   r,
		RawBody:   summary,
	})
}
