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

func GetPromotionsTypesHandler(c *gin.Context) {
	start := time.Now()

	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	promotions, err := models.GetPromotionsTypes()
	if err != nil {
		log.Printf("promotion types error::%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to fetch promotion types",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to fetch promotion types",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Respond
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promotion types fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   promotions,
		Message:   "Promotion types fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// CreatePromotionsTypesHandler handles POST /api/promotions/types
//
// @Summary      Create promotions types
// @Description  create promotions types
// @Tags         Promotions
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  dtos.ErrorResponse
// @Router       /api/home/promotions/types [post]
func CreatePromotionsTypesHandler(c *gin.Context) {
	start := time.Now()

	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "promotions.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	req, ok := DecodeRequestBody[dtos.PromotionType](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Promotions") {
		return
	}
	err := models.CreatePromotionType(models.DB, *req)
	if err != nil {
		log.Printf("promotion types error::%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to create promotion type",
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

	// Respond
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promotion type created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Promotion type created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// EditPromotionsTypesHandler handles PATCH /api/promotions/types/{id}
//
// @Summary      Update promotions types
// @Description  update promotions types
// @Tags         Promotions
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  dtos.ErrorResponse
// @Router       /api/home/promotions/types/{id} [patch]
func UpdatePromotionsTypesHandler(c *gin.Context) {
	start := time.Now()

	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "promotions.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	req, ok := DecodeRequestBody[dtos.PromotionType](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Promotions") {
		return
	}
	typeID := c.Param("id")
	err := models.UpdatePromotionType(models.DB, typeID, *req)
	if err != nil {
		log.Printf("promotion types error::%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to update promotion type",
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

	// Respond
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promotion type updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Promotion type updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// DeletePromotionsTypesHandler handles DELETE /api/promotions/types/{id}
//
// @Summary      Delete promotions types
// @Description  delete promotions types
// @Tags         Promotions
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  dtos.ErrorResponse
// @Router       /api/home/promotions/types/{id} [delete]
func DeletePromotionsTypesHandler(c *gin.Context) {
	start := time.Now()

	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify user has admin privileges (required for media uploads)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "promotions.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	typeID := c.Param("id")
	err := models.DeletePromotionType(typeID)
	if err != nil {
		log.Printf("promotion types error::%s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to delete promotion type",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to delete promotion type",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Respond
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promotion type deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Promotion type deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// NewPromotionHandler creates a new promotion.
// This endpoint is restricted to administrators.
//
// @Summary      Create new promotions
// @Description  Create Promotion
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        promotion  body      dtos.NewPromotion  true  "Promotion Details"
// @Success      201        {object}  map[string]any
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/promotions [post]
func NewPromotionHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Promotions", "promotions.create")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.NewPromotion](c, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Promotions") {
		return
	}
	if !req.StartDate.Before(req.EndDate) {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Invalid validity period when creating promotion",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid validity period",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	_, err := models.CreateNewPromotion(*req)
	if err != nil {
		log.Printf("Error creating promotion: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to add promotion",
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
			Description: "Promotion created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Promotion created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeletePromotionHandler deletes a promotion.
// This endpoint is restricted to administrators.
//
// @Summary      Delete a promotion
// @Description  Deletes a promotion
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        promotion_id  path      string  true  "Promotion ID"
// @Success      200           {object}  map[string]any
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      401           {object}  dtos.ErrorResponse
// @Failure      500           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/promotions/{promotion_id} [delete]
