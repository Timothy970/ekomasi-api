package utils

import (
	"ekomasi_backend/middleware"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

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

// RequirePermissions verifies that the user is authenticated, not a customer and has one of the allowed permissions.
//
// This function:
// 1. Extracts authenticated user from request context
// 2. Verifies user is authenticated (sends 403 if not)
// 3. Verifies user has one of the allowed permissions (sends 403 if not)
// 4. Returns user and success status
//
// Use this for endpoints that require specific permission.
//
// Parameters:
//   - r: *http.Request - HTTP request with user context
//   - w: http.ResponseWriter - Response writer for error responses
//   - start: time.Time - Request start time for performance tracking
//   - requestSummary: string - Request summary for logging
//   - module: string - Module name for error context
//   - allowedPermissions: []string - List of permissions that are permitted to access
//
// Returns:
//   - middleware.AuthenticatedUser: User data if authorized (empty if not)
//   - bool: true if user has allowed permission, false otherwise (response sent)
var RequirePermissions = func(
	r *http.Request,
	w http.ResponseWriter,
	start time.Time,
	requestSummary string,
	module string,
	allowedPermission string,
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

	//check user is not a customer(role)
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

	// Admin role has all permissions, skip permission check
	if user.Role == "admin" {
		return user, true
	}
	//if allowedPermission passed is empty, skip permission check
	if allowedPermission == "" {
		return user, true
	}
	//for roles like manager, marketing, customer-service, check permissions
	// Check if user's role is in the allowed roles list
	hasPermission := false
	userPermissions := user.Permissions
	for _, userPermission := range userPermissions {
		if strings.EqualFold(userPermission, allowedPermission) {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		availablePermissions := SupportedPermissions
		errorMsg := "You don't have permission to perform this action"
		//get the description of the needed permission
		for _, perm := range availablePermissions {
			if strings.EqualFold(perm.Key, allowedPermission) {
				errorMsg = fmt.Sprintf("You don't have permission to %s", strings.ToLower(perm.Description))
				break
			}
		}

		// User doesn't have required role - send 403 Forbidden
		RespondWithError(w, ErrorJSONResponseOptions{
			CollectiveInfo: CollectiveInfo{
				Module:      module,
				Description: "Authorization failed",
				Code:        http.StatusForbidden,
			},
			Message:   errorMsg,
			TimeTaken: time.Since(start),
			Function:  GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return middleware.AuthenticatedUser{}, false
	}

	// User is authenticated and has one of the allowed roles
	return user, true
}

// RequireGinPermissions verifies authorization using native Gin context
func RequireGinPermissions(
	c *gin.Context,
	start time.Time,
	requestSummary string,
	module string,
	allowedPermission string,
) (middleware.AuthenticatedUser, bool) {
	return RequirePermissions(c.Request, c.Writer, start, requestSummary, module, allowedPermission)
}

// ValidateGinStructAndRespond validates struct fields and responds via Gin if invalid
