package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/csv"
)

func DownloadUsersCSVHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if the requesting user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Users", ""); !ok {
		return
	}

	// Parse filters (ignore pagination for CSV export)
	q := c.Query("q")
	role := c.Query("role")
	isAdmin := c.Query("isAdmin")

	// Fetch users - using a large limit for export
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	users, _, err := models.GetAllUsersWithPagination(models.DB, 1000000, 0, q, role, isAdmin, tenantID)
	if err != nil {
		log.Printf("Error fetching users for CSV: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch users for CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Set headers for CSV download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=users.csv")

	// Initialize CSV writer
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header row
	header := []string{"First Name", "Last Name", "Email", "Phone", "Role", "Status", "Date Joined", "Last Login"}
	if err := writer.Write(header); err != nil {
		log.Printf("Error writing CSV header: %v", err)
		return
	}

	// Write data rows
	for _, u := range users {
		row := []string{
			u.FirstName,
			u.LastName,
			u.Email,
			u.Phone,
			u.Role,
			u.Status,
			u.DateJoined,
			u.LastLogin,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Error writing CSV row: %v", err)
			return
		}
	}
}
