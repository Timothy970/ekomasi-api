package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateBlogHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "promotions.create")
	if !ok {
		return
	}
	authUser, userOk := middleware.UserFromContext(c.Request.Context())
	if !userOk || authUser.Role != "admin" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "User is not authorized to create blog",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User is not authorized",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}
	// url, err := utils.ParseAndUploadFile(c.Request, "image", 20)
	// if err != nil {
	// 	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
	// 		CollectiveInfo: utils.CollectiveInfo{
	// 			Module:      "Homepage",
	// 			Description: "Failed to upload file when creating blog: " + err.Error(),
	// 			Code:        http.StatusBadRequest,
	// 		},
	// 		Message:   "Failed to upload file: " + err.Error(),
	// 		TimeTaken: time.Since(start),
	// 		Function:  utils.GetCurrentFuncName(),
	// 		Request: c.Request,
	// 	})
	// 	return
	// }
	// author := c.Request.FormValue("author")
	// blog := &dtos.Blog{
	// 	ImageURL: &url,
	// 	Title:    c.Request.FormValue("title"),
	// 	Content:  c.Request.FormValue("content"),
	// 	Author:   &author,
	// 	AuthorID: authUser.ID,
	// }
	blog, ok := DecodeRequestBody[dtos.BlogRequest](c, requestSummary, start)
	if !ok {
		return
	}

	if blog.ReadTimeMinutes == 0 {
		blog.ReadTimeMinutes = 5 //default read time
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(blog, c, requestSummary, start, "Homepage") {
		return
	}
	if err := models.CreateBlog(*blog, authUser.ID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to create blog: " + err.Error(),
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
			Description: "Blog created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Blog created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetBlogHandler retrieves a blog by ID.
//
// @Summary      Get blog by ID
// @Description  Get blog by ID
// @Tags         Blogs
// @Produce      json
// @Param        blog_id  path      string  true  "Blog ID"
// @Success      200      {object}  dtos.Blog
// @Failure      500      {object}  dtos.ErrorResponse
// @Router       /api/admin/blogs/{blog_id} [get]
func GetBlogHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	blogID := c.Param("blog_id")
	blog, err := models.GetBlogByID(blogID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to get blog by ID: " + err.Error(),
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
			Description: blogWithID + blogID + " fetched successfully",
			Code:        http.StatusCreated,
		},
		Payload:   blog,
		Message:   "Blog fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateBlogHandler updates an existing blog post.
// This endpoint is restricted to administrators.
//
// @Summary      Update blog
// @Description  Update blog
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        blog_id  path      string            true  "Blog ID"
// @Param        blog     body      dtos.BlogRequest  true  "Blog Details"
// @Success      200      {object}  map[string]any
// @Failure      500      {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/blogs/{blog_id} [patch]
func UpdateBlogHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "promotions.update")
	if !ok {
		return
	}
	blog, ok := DecodeRequestBody[dtos.BlogRequest](c, requestSummary, start)
	if !ok {
		return
	}

	if blog.ReadTimeMinutes == 0 {
		blog.ReadTimeMinutes = 5 //default read time
	}

	//Validate the request
	if !utils.ValidateGinStructAndRespond(blog, c, requestSummary, start, "Homepage") {
		return
	}
	blogID := c.Param("blog_id")
	if err := models.UpdateBlog(*blog, blogID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to update blog with ID " + blogID + ": " + err.Error(),
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
			Description: blogWithID + blogID + " has been updated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Blog updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteBlogHandler deletes a blog post.
// This endpoint is restricted to administrators.
//
// @Summary      Delete blog
// @Description  Delete blog
// @Tags         Admin
// @Produce      json
// @Param        blog_id  path      string  true  "Blog ID"
// @Success      200      {object}  map[string]any
// @Failure      500      {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/blogs/{blog_id} [delete]
func DeleteBlogHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Homepage", "promotions.delete")
	if !ok {
		return
	}
	blogID := c.Param("blog_id")
	if err := models.DeleteBlog(blogID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Homepage",
				Description: "Failed to delete blog with ID " + blogID + ": " + err.Error(),
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
			Description: blogWithID + blogID + " was deleted successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Blog deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListBlogsHandler retrieves a paginated list of blogs.
//
// @Summary      List blogs
// @Description  List blogs
// @Tags         Admin
// @Produce      json
// @Param        page    query     int     false  "Page number"
// @Param        size    query     int     false  "Page size"
// @Param        status  query     string  false  "Blog Status"
// @Success      200     {object}  map[string]any
// @Failure      500     {object}  dtos.ErrorResponse
// @Router       /api/admin/blogs [get]
