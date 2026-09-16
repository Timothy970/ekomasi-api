package handlers

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListBlogsHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	title := c.Query("title")
	isAdmin := false
	admin := c.Query("isAdmin")
	if admin != "" {
		isAdmin = true
	}
	blogs, pagination, err := models.ListBlogs(page, limit, title, isAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Homepage",
					Description: "No blogs found",
					Code:        http.StatusNotFound,
				},
				Message:   "Blogs not found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
		} else {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Homepage",
					Description: "Failed to list blogs: " + err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
		}
		return
	}
	response := map[string]any{
		"blogs":      blogs,
		"pagination": pagination,
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Blogs fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Blogs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// CreateMenuLink creates a new menu link.
// This endpoint is restricted to administrators.
//
// @Summary      Create Menu links
// @Description  Create Menu Links
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        menu_link  body      dtos.MenuLinkRequest  true  "Menu Link Details"
// @Success      200        {object}  map[string]any
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/menu-links [post]
func CreateMenuLink(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.MenuLinkRequest](c, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Homepage") {
		return
	}

	err := models.CreateMenuLink(*req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to create menu link: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Menu link created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Menu link created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateMenuLink updates an existing menu link.
// This endpoint is restricted to administrators.
//
// @Summary      Update Menu links
// @Description  Update Menu Links
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        menulink_id  path      string                true  "Menu Link ID"
// @Param        menu_link    body      dtos.MenuLinkRequest  true  "Menu Link Details"
// @Success      200          {object}  map[string]any
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/menu-links/{menulink_id} [patch]
func UpdateMenuLink(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.MenuLinkRequest](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Homepage") {
		return
	}
	menulinkID, _ := strconv.Atoi(c.Param("menulink_id"))
	err := models.UpdateMenuLink(*req, menulinkID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to update menu link with ID " + strconv.Itoa(menulinkID) + ": " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Menu link with ID " + strconv.Itoa(menulinkID) + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Menu link updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteMenuLink deletes a menu link.
// This endpoint is restricted to administrators.
//
// @Summary      Delete Menu links
// @Description  Delete Menu Links
// @Tags         Admin
// @Produce      json
// @Param        menulink_id  path      string  true  "Menu Link ID"
// @Success      200          {object}  map[string]any
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/menu-links/{menulink_id} [delete]
func DeleteMenuLink(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "")
	if !ok {
		return
	}
	id, _ := strconv.Atoi(c.Param("menulink_id"))
	err := models.DeleteMenuLink(id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to delete menu link with ID " + strconv.Itoa(id) + ": " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Menu link with ID " + strconv.Itoa(id) + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Menu link deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// CreateSocialLinkHandler creates a new social link.
// This endpoint is restricted to administrators.
//
// @Summary      Create Social
// @Description  Create Social
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        social_link  body      dtos.SocialLinkRequest  true  "Social Link Details"
// @Success      200          {object}  map[string]any
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/socials [post]
func CreateSocialLinkHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.SocialLinkRequest](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Homepage") {
		return
	}
	if err := models.CreateSocialLink(req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to create social link: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Homepage",
			Description: "Social link created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Social added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
