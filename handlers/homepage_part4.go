package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func DeletePromotionHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "promotions.delete")
	if !ok {
		return
	}
	promotionID := c.Param("promotion_id")
	err := models.DeletePromotion(promotionID)
	if err != nil {
		log.Printf("Error deleting promotion: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to delete promotion with ID " + promotionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	_ = utils.DeleteCache("promotions")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promotion with ID " + promotionID + " has been deleted successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Promotion deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// EditPromotionHandler edits an existing promotion.
// This endpoint is restricted to administrators.
//
// @Summary      Edit a promotion
// @Description  Edit a promotion
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        promotion_id  path      string              true  "Promotion ID"
// @Param        promotion     body      dtos.EditPromotion  true  "Promotion Details"
// @Success      200           {object}  map[string]any
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      401           {object}  dtos.ErrorResponse
// @Failure      500           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/promotions/{promotion_id} [patch]
func EditPromotionHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "promotions.update")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.EditPromotion](c, requestSummary, start)
	if !ok {
		return
	}
	err := models.EditPromotion(*req)
	if err != nil {
		log.Printf("Error editing promotion: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to edit promotion",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	_ = utils.DeleteCache("promotions")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promotion edited successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Promotion edited successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AttachProductToPromotionHandler attaches a product to a promotion.
// This endpoint is restricted to administrators.
//
// @Summary      Attach a product to promotion
// @Description  Attach a product to promotion
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        promotion_id  path      string                       true  "Promotion ID"
// @Param        attachment    body      dtos.AttachProductToPromotion true  "Attachment Details"
// @Success      201           {object}  map[string]any
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      401           {object}  dtos.ErrorResponse
// @Failure      500           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/promotions/{promotion_id} [post]
func AttachProductToPromotionHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "products.create")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.AttachProductToPromotion](c, requestSummary, start)
	if !ok {
		return
	}

	//check if promotion exists
	ok = checkIfPromotionExists(req.PromotionID, c, start)
	if !ok {
		return
	}
	err := models.AttachProductsToPromotion(*req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to add products to promotion with ID " + req.PromotionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	_ = utils.DeleteCache("promotions")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Products added to promotion with ID " + req.PromotionID + " successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Products added to promotion successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// RemoveProductFromPromotionHandler removes a product from a promotion.
// This endpoint is restricted to administrators.
//
// @Summary      Remove product from promotion
// @Description  Remove a product from a promotion
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        promotion_id  path      string                       true  "Promotion ID"
// @Param        attachment    body      dtos.AttachProductToPromotion true  "Attachment Details"
// @Success      200           {object}  map[string]any
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      500           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/promotions/{promotion_id}/products [delete]
func RemoveProductFromPromotionHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	req, ok := DecodeRequestBody[dtos.AttachProductToPromotion](c, requestSummary, start)
	if !ok {
		return
	}
	// user, ok := middleware.UserFromContext(c.Request.Context())
	// if !ok {
	// 	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
	// 		Code:      http.StatusInternalServerError,
	// 		Message:   noUser,
	// 		TimeTaken: time.Since(start),
	// 		Function:  utils.GetCurrentFuncName(),
	// 		Request: c.Request,
	// 		RawBody:   requestSummary})
	// 	return
	// }
	//check if promotion exists
	ok = checkIfPromotionExists(req.PromotionID, c, start)
	if !ok {
		return
	}
	err := models.RemoveProductFromPromotion(*req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to remove product from promotion with ID " + req.PromotionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	_ = utils.DeleteCache("promotions")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Products added to promotion with ID " + req.PromotionID + " successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Products added to promotion successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
func checkIfPromotionExists(promotionID string, c *gin.Context, start time.Time) bool {
	promotionExists, err := models.CheckPromotionExists(promotionID)
	if err != nil || !promotionExists {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Promotion does not exist",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Promotion does not exist",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return false
	}
	return true
}

// CreateBlogHandler creates a new blog post.
// This endpoint is restricted to administrators.
//
// @Summary      Create new blog
// @Description  Create blog
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        blog  body      dtos.BlogRequest  true  "Blog Details"
// @Success      200   {object}  map[string]any
// @Failure      500   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/blogs [post]
