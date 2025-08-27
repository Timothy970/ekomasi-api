package utils

import (
	"adenzo_backend/middleware"
	"fmt"
	"net/http"
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
func ValidateStructAndRespond(
	data any,
	w http.ResponseWriter,
	r *http.Request,
	requestSummary string,
	start time.Time,
) bool {
	validationErrors := ValidateStruct(data)

	if len(validationErrors) > 0 {
		RespondWithJSON(w, SuccessJSONResponseOptions{
			Code:      http.StatusBadRequest,
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

func RequireAdmin(
	r *http.Request,
	w http.ResponseWriter,
	start time.Time,
	requestSummary string,
) (middleware.AuthenticatedUser, bool) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		RespondWithError(w, ErrorJSONResponseOptions{
			Code:      http.StatusForbidden,
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
			Code:      http.StatusForbidden,
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
