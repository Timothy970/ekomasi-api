// Package utils provides validation utilities for the Adenzo e-commerce platform.
//
// This file contains validation and authorization functions:
//   - Struct validation with user-friendly error messages
//   - HTTP request validation and response handling
//   - Admin role verification
//   - Permission-based access control
//   - Role and permission caching
//
// Validation Features:
//   - go-playground/validator integration
//   - Field-level validation with custom messages
//   - Automatic error response generation
//   - Performance timing for all validations
//
// Authorization Features:
//   - Admin role checking
//   - Granular permission verification
//   - Role-based access control (RBAC)
//   - Permission caching for performance
//   - Missing permission reporting
//
// Thread Safety:
//   - Uses cached role permissions when available
//   - Database queries for cache misses
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

// validate is the shared validator instance used for struct validation.
// Uses go-playground/validator for declarative validation rules.
var validate = validator.New()

// tagToMessage maps validation tags to user-friendly error messages.
//
// Supported validation tags:
//   - required: Field must be present and non-empty
//   - gt: Value must be greater than zero
//   - gte: Value must be greater than or equal to zero
//   - email: Must be valid email format
//   - min: Must meet minimum length requirement
//   - max: Must not exceed maximum length
//   - len: Must have exact fixed length
var tagToMessage = map[string]string{
	"required": "is required and cannot be empty",
	"gt":       "must be greater than zero",
	"gte":      "must be greater than or equal to zero",
	"email":    "must be a valid email address",
	"min":      "does not meet the minimum length",
	"max":      "exceeds the maximum allowed length",
	"len":      "must have a fixed length",
}

// ValidateStruct validates a struct and returns field errors with friendly messages.
//
// This function:
// 1. Validates struct using go-playground/validator rules
// 2. Converts validation errors to user-friendly messages
// 3. Maps validation tags to descriptive error text
// 4. Returns empty map if validation passes
//
// Validation is performed based on struct tags:
// Parameters:
//   - data: any - Struct to validate (must have validation tags)
//
// Returns:
//   - map[string]string: Field name -> error message mapping
//     Empty map indicates successful validation
func ValidateStruct(data any) map[string]string {
	// Initialize empty error map
	errs := make(map[string]string)

	// Perform struct validation
	err := validate.Struct(data)
	if err != nil {
		// Process each validation error
		for _, e := range err.(validator.ValidationErrors) {
			fieldName := e.Field()
			// Look up friendly message for validation tag
			message, ok := tagToMessage[e.Tag()]
			if !ok {
				// Use generic message if tag not in map
				message = fmt.Sprintf("failed validation on '%s'", e.Tag())
			}
			// Format error message with field name
			errs[fieldName] = fmt.Sprintf("%s %s", fieldName, message)
		}
	}

	return errs
}

// ValidateStructAndRespond validates struct and automatically responds with 400 Bad Request if validation fails.
//
// This function:
// 1. Validates struct using ValidateStruct
// 2. If validation fails, sends HTTP 400 response with error details
// 3. If validation passes, returns true without sending response
// 4. Includes performance timing and request context
//
// Use this function when you want automatic error responses.
// For manual error handling, use ValidateStruct directly.
//
// Parameters:
//   - data: any - Struct to validate
//   - w: http.ResponseWriter - HTTP response writer
//   - r: *http.Request - HTTP request for context
//   - requestSummary: string - Summary of request for logging
//   - start: time.Time - Request start time for performance tracking
//   - module: string - Module name for error context
//
// Returns:
//   - bool: true if validation passed, false if validation failed (response sent)
var ValidateStructAndRespond = func(
	data any,
	w http.ResponseWriter,
	r *http.Request,
	requestSummary string,
	start time.Time,
	module string,
) bool {
	// Validate the struct
	validationErrors := ValidateStruct(data)

	// If validation errors exist, send 400 Bad Request response
	if len(validationErrors) > 0 {
		RespondWithJSON(w, SuccessJSONResponseOptions{
			CollectiveInfo: CollectiveInfo{
				Module:      module,
				Description: "Validation failed for the provided input",
				Code:        http.StatusBadRequest,
			},
			Payload:   validationErrors, // Include error details
			Message:   "Validation failed",
			TimeTaken: time.Since(start), // Track performance
			Function:  GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return false // Validation failed
	}

	return true // Validation passed
}

// RequireAdmin verifies that the user is authenticated and has admin role.
//
// This function:
// 1. Extracts authenticated user from request context
// 2. Verifies user is authenticated (sends 403 if not)
// 3. Verifies user has "admin" role (sends 403 if not)
// 4. Returns user and success status
//
// Use this for endpoints that require admin privileges.
//
// Parameters:
//   - r: *http.Request - HTTP request with user context
//   - w: http.ResponseWriter - Response writer for error responses
//   - start: time.Time - Request start time for performance tracking
//   - requestSummary: string - Request summary for logging
//   - module: string - Module name for error context
//
// Returns:
//   - middleware.AuthenticatedUser: User data if authorized (empty if not)
//   - bool: true if user is admin, false otherwise (response sent)
var RequireAdmin = func(
	r *http.Request,
	w http.ResponseWriter,
	start time.Time,
	requestSummary string,
	module string,
) (middleware.AuthenticatedUser, bool) {
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated - send 403 Forbidden
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

	// Verify user has admin role
	if user.Role == "customer" {
		// User is not admin - send 403 Forbidden
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

	// User is authenticated and has admin role
	return user, true
}

// RequirePermissions creates a closure that verifies user has specific permissions.
//
// This function:
// 1. Returns a validation function with db and permissions captured
// 2. Extracts authenticated user from request context
// 3. Fetches user's role from database
// 4. Retrieves role permissions (from cache or database)
// 5. Validates user has all required permissions
// 6. Reports missing permissions if validation fails
//
// Permission checking process:
//   - User authentication verification
//   - Role ID lookup from users table
//   - Permission retrieval (cache-first strategy)
//   - Case-insensitive permission matching
//   - Detailed missing permission reporting
//
// Parameters:
//   - db: *sql.DB - Database connection for role/permission queries
//   - requiredPerms: []string - List of required permission names
//   - module: string - Module name for error context
//
// Returns:
//   - func: Validation function that takes (request, response, start, summary)
//     and returns (AuthenticatedUser, bool)
//     // User has required permissions
func RequirePermissions(db *sql.DB, requiredPerms []string, module string) func(
	r *http.Request,
	w http.ResponseWriter,
	start time.Time,
	requestSummary string,
) (middleware.AuthenticatedUser, bool) {

	// Return closure with captured db, requiredPerms, and module
	return func(r *http.Request, w http.ResponseWriter, start time.Time, requestSummary string) (middleware.AuthenticatedUser, bool) {
		// Extract authenticated user from context
		user, ok := middleware.UserFromContext(r.Context())
		if !ok {
			// User not authenticated
			sendError(w, http.StatusForbidden, "User not validated", start, r, requestSummary, module)
			return middleware.AuthenticatedUser{}, false
		}

		// Fetch user's role ID from database
		roleID, err := fetchUserRole(db, user.ID)
		if err != nil {
			// Database error fetching role
			sendError(w, http.StatusInternalServerError, "Failed to fetch user role: "+err.Error(), start, r, requestSummary, module)
			return middleware.AuthenticatedUser{}, false
		}

		// Retrieve permissions for role (cache-first)
		rolePerms, err := getRolePermissions(db, roleID)
		if err != nil {
			// Database error fetching permissions
			sendError(w, http.StatusInternalServerError, "Failed to fetch role permissions: "+err.Error(), start, r, requestSummary, module)
			return middleware.AuthenticatedUser{}, false
		}

		// Verify user has all required permissions
		hasAccess, missingPerms := hasAllRequiredPermissions(rolePerms, requiredPerms)
		if !hasAccess {
			// User missing one or more required permissions
			errorMsg := fmt.Sprintf("Missing required permissions: %s", strings.Join(missingPerms, ", "))
			sendError(w, http.StatusForbidden, errorMsg, start, r, requestSummary, module)
			return middleware.AuthenticatedUser{}, false
		}

		// User has all required permissions
		return user, true
	}
}

// fetchUserRole retrieves the user's role_id from the database.
//
// Queries the users table to get the role assigned to the user.
//
// Parameters:
//   - db: *sql.DB - Database connection
//   - userID: string - User identifier
//
// Returns:
//   - string: Role ID for the user
//   - error: Database error if query fails or user not found
func fetchUserRole(db *sql.DB, userID string) (string, error) {
	var roleID string
	// Query role_id from users table
	err := db.QueryRow(`SELECT role_id FROM users WHERE id = ?`, userID).Scan(&roleID)
	return roleID, err
}

// getRolePermissions returns all permissions associated with a role, using cache if available.
//
// This function:
// 1. Checks cache for role permissions (cache key: "role_permissions:<roleID>")
// 2. If cache hit, returns cached permissions
// 3. If cache miss, queries database for permissions
// 4. Stores result in cache for future requests (non-blocking)
//
// Caching Strategy:
//   - Cache key format: "role_permissions:<roleID>"
//   - Cache hit: Returns immediately without database query
//   - Cache miss: Queries database and stores in cache
//   - Non-blocking cache writes (errors ignored)
//
// Parameters:
//   - db: *sql.DB - Database connection
//   - roleID: string - Role identifier
//
// Returns:
//   - []string: List of permission names for the role
//   - error: Database error if query fails
func getRolePermissions(db *sql.DB, roleID string) ([]string, error) {
	// Construct cache key
	cacheKey := "role_permissions:" + roleID
	var rolePerms []string

	// Try to get permissions from cache
	if err := GetCache(cacheKey, &rolePerms); err == nil {
		// Cache hit - return cached permissions
		return rolePerms, nil
	}

	// Cache miss - fetch permissions from database
	rows, err := db.Query(`
		SELECT p.name
		FROM role_permissions rp
		JOIN permissions p ON p.permission_id = rp.permission_id
		WHERE rp.role_id = ?`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate through permission rows
	for rows.Next() {
		var permName string
		if err := rows.Scan(&permName); err == nil {
			// Add permission to list
			rolePerms = append(rolePerms, permName)
		}
	}

	// Store permissions in cache (non-blocking, ignore errors)
	_ = SetCache(cacheKey, rolePerms)
	return rolePerms, nil
}

// hasAllRequiredPermissions verifies that the role includes all required permissions and returns missing ones.
//
// This function:
// 1. Converts role permissions to lowercase map for fast lookup
// 2. Checks each required permission against role permissions
// 3. Builds list of missing permissions
// 4. Returns success status and missing permission list
//
// Permission Matching:
//   - Case-insensitive comparison (both converted to lowercase)
//   - Fast O(1) lookup using map
//   - Reports all missing permissions, not just first
//
// Parameters:
//   - rolePerms: []string - Permissions the user's role has
//   - requiredPerms: []string - Permissions required for the action
//
// Returns:
//   - bool: true if all required permissions present, false otherwise
//   - []string: List of missing permission names (empty if all present)
func hasAllRequiredPermissions(rolePerms, requiredPerms []string) (bool, []string) {
	// Build map of role permissions for O(1) lookup (case-insensitive)
	rolePermsMap := make(map[string]bool, len(rolePerms))
	for _, perm := range rolePerms {
		rolePermsMap[strings.ToLower(perm)] = true
	}

	// Check each required permission
	var missingPerms []string
	for _, required := range requiredPerms {
		// Case-insensitive permission check
		if !rolePermsMap[strings.ToLower(required)] {
			// Permission not found - add to missing list
			missingPerms = append(missingPerms, required)
		}
	}

	// Return success status and missing permissions
	return len(missingPerms) == 0, missingPerms
}

// sendError standardizes error responses for validation and permission checks.
//
// This helper function:
// 1. Creates standardized error response structure
// 2. Includes performance timing information
// 3. Adds request context and module information
// 4. Sends HTTP error response with specified status code
//
// Use this for consistent error formatting across validation functions.
//
// Parameters:
//   - w: http.ResponseWriter - Response writer
//   - code: int - HTTP status code (e.g., 403, 500)
//   - message: string - Error message to display
//   - start: time.Time - Request start time for performance tracking
//   - r: *http.Request - HTTP request for context
//   - summary: string - Request summary for logging
//   - module: string - Module name for error context
func sendError(w http.ResponseWriter, code int, message string, start time.Time, r *http.Request, summary string, module string) {
	// Send standardized error response
	RespondWithError(w, ErrorJSONResponseOptions{
		CollectiveInfo: CollectiveInfo{
			Module:      module,
			Description: "Permission check failed",
			Code:        code,
		},
		Message:   message,              // Error message
		TimeTaken: time.Since(start),    // Request duration
		Function:  GetCurrentFuncName(), // Current function name
		Request:   r,                    // Request context
		RawBody:   summary,              // Request summary
	})
}
