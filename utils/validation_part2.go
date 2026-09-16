package utils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func ValidateGinStructAndRespond(
	data any,
	c *gin.Context,
	requestSummary string,
	start time.Time,
	module string,
) bool {
	errs := ValidateStruct(data)
	if len(errs) > 0 {
		RespondWithGinError(c, ErrorJSONResponseOptions{
			CollectiveInfo: CollectiveInfo{
				Module:      module,
				Description: "Validation failed",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("Validation failed: %v", errs),
			TimeTaken: time.Since(start),
			Function:  GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return false
	}
	return true
}
