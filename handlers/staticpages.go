// Package handlers provides HTTP request handlers for static page management.
// This file contains handlers for CMS (Content Management System) functionality,
// allowing admins to create, update, retrieve, and delete static pages like About Us,
// Terms & Conditions, Privacy Policy, FAQs, and other informational content.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// CreateStaticPage creates a new static content page in the CMS.
// Admin-only operation for adding informational pages like About, Terms, Privacy Policy, etc.
// Associates the page with the creator's user ID for audit tracking.
//
// @Summary      Create static page
// @Description  Create a new static content page (admin only)
// @Tags         Static Pages
// @Accept       json
// @Produce      json
// @Param        page  body      dtos.StaticPageRequest   true  "Static page content"
// @Success      201   {object}  dtos.SuccessResponse     "Page created successfully"
// @Failure      400   {object}  dtos.ErrorResponse       "Invalid request or validation failed"
// @Failure      401   {object}  dtos.ErrorResponse       "Admin authorization required"
// @Failure      404   {object}  dtos.ErrorResponse       "Failed to create page"
// @Security     BearerAuth
// @Router       /api/admin/static-pages [post]
func CreateStaticPage(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can create static pages)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "HomePage")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract authenticated user from request context for audit tracking
	authuser, _ := middleware.UserFromContext(r.Context())

	// Decode and parse JSON request body with page details
	req, ok := DecodeRequestBody[dtos.StaticPageRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (title, slug, content, etc.)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "HomePage") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Create new static page in database with creator's user ID
	err := models.CreateStaticPage(*req, authuser.ID)
	if err != nil {
		// Page creation failed (duplicate slug, database error, etc.)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to create static page" + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Static page created successfully
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusCreated,
			Description: "Static page created successfully",
		},
		Payload:   nil,
		Message:   "Static page created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// GetStaticPages retrieves all static pages with optional search filtering.
// Returns a list of static pages, optionally filtered by search query.
// Public endpoint for displaying informational pages on the website.
//
// @Summary      List static pages
// @Description  Retrieve all static pages with optional search query
// @Tags         Static Pages
// @Produce      json
// @Param        q  query     string                 false  "Search query for filtering pages"
// @Success      200         {array}   dtos.StaticPage       "Static pages list"
// @Failure      404         {object}  dtos.ErrorResponse    "Failed to fetch pages"
// @Router       /api/static-pages [get]
func GetStaticPages(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract optional search query parameter
	query := r.URL.Query().Get("q")
	// Fetch static pages from database, filtered by search query if provided
	staticPages, err := models.GetStaticPages(query)
	if err != nil {
		// Database query failed or no pages found
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to fetch static pages" + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return list of static pages for website display
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusOK,
			Description: "Static pages fetched successfully",
		},
		Payload:   staticPages,
		Message:   "Static pages fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})

}

// GetStaticPageByID retrieves a specific static page by its unique identifier.
// Returns detailed page content including title, slug, body, meta tags, and timestamps.
// Public endpoint for rendering individual static pages.
//
// @Summary      Get static page by ID
// @Description  Retrieve a specific static page's full content
// @Tags         Static Pages
// @Produce      json
// @Param        static_page_id  path      string              true  "Static page ID"
// @Success      200             {object}  dtos.StaticPage     "Static page details"
// @Failure      404             {object}  dtos.ErrorResponse  "Page not found"
// @Router       /api/static-pages/{static_page_id} [get]
func GetStaticPageByID(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract static page ID from URL path parameters
	staticPageID := mux.Vars(r)["static_page_id"]
	// Fetch specific static page from database by ID
	staticPage, err := models.GetStaticPageByID(staticPageID)
	if err != nil {
		// Page not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to fetch static page with ID " + staticPageID + ": " + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return static page content for rendering
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusOK,
			Description: "Static page with ID" + staticPageID + " fetched successfully",
		},
		Payload:   staticPage,
		Message:   "Static page fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// DeleteStaticPage permanently removes a static page from the CMS.
// Admin-only operation for removing outdated or unwanted informational pages.
// Deletion is permanent and cannot be undone - use with caution.
//
// @Summary      Delete static page
// @Description  Permanently remove a static page (admin only)
// @Tags         Static Pages
// @Produce      json
// @Param        static_page_id  path      string                true  "Static page ID"
// @Success      200             {object}  dtos.SuccessResponse  "Page deleted successfully"
// @Failure      401             {object}  dtos.ErrorResponse    "Admin authorization required"
// @Failure      404             {object}  dtos.ErrorResponse    "Page not found or deletion failed"
// @Security     BearerAuth
// @Router       /api/admin/static-pages/{static_page_id} [delete]
func DeleteStaticPage(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete static pages)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "HomePage")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract static page ID from URL path parameters
	staticPageID := mux.Vars(r)["static_page_id"]
	// Permanently delete static page from database
	err := models.DeleteStaticPage(staticPageID)
	if err != nil {
		// Deletion failed (page not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to delete static page with ID " + staticPageID + ": " + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Static page deleted successfully
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusOK,
			Description: "Static page with ID " + staticPageID + " deleted successfully",
		},
		Payload:   nil,
		Message:   "Static page deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// UpdateStaticPage modifies an existing static page's content.
// Admin-only operation for updating page title, body, slug, meta tags, etc.
// Essential for maintaining current and accurate informational content.
//
// @Summary      Update static page
// @Description  Update existing static page content (admin only)
// @Tags         Static Pages
// @Accept       json
// @Produce      json
// @Param        static_page_id  path      string                  true  "Static page ID"
// @Param        page            body      dtos.StaticPageRequest  true  "Updated page content"
// @Success      200             {object}  dtos.StaticPage         "Updated page details"
// @Failure      400             {object}  dtos.ErrorResponse      "Invalid request or validation failed"
// @Failure      401             {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      404             {object}  dtos.ErrorResponse      "Page not found or update failed"
// @Security     BearerAuth
// @Router       /api/admin/static-pages/{static_page_id} [put]
func UpdateStaticPage(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update static pages)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "HomePage")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract static page ID from URL path parameters
	staticPageID := mux.Vars(r)["static_page_id"]
	// Decode and parse JSON request body with updated page content
	req, ok := DecodeRequestBody[dtos.StaticPageRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the update request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "HomePage") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Update static page in database with new content
	staticPage, err := models.UpdateStaticPage(staticPageID, *req)
	if err != nil {
		// Update failed (page not found, duplicate slug, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to update static page with ID " + staticPageID + ": " + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Return updated static page content
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusOK,
			Description: "Static page with ID " + staticPageID + " updated successfully",
		},
		Payload:   staticPage,
		Message:   "Static page updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
