package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var chargeWithID = "Charge with ID "

// Add Charge
// AddChargeHandler creates a new charge.
// This endpoint is restricted to administrators.
//
// @Summary      Add a new charge
// @Description  Create a new charge
// @Tags         Charges
// @Accept       json
// @Produce      json
// @Param        charge  body      dtos.Charge  true  "Charge Details"
// @Success      201     {object}  dtos.Charge
// @Failure      400     {object}  dtos.ErrorResponse
// @Failure      500     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/charges [post]
func AddChargeHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Charges", "charges.create"); !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.Charge](c, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Charges") {
		return
	}

	charge, err := models.AddCharge(models.DB, *req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Charges",
				Description: "Failed to add charge",
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
			Module:      "Charges",
			Description: "Charge added successfully",
			Code:        http.StatusCreated,
		},
		Payload: charge, Message: "Charge added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
		RawBody:  requestSummary})
}

// Update Charge
// UpdateChargeHandler updates an existing charge.
// This endpoint is restricted to administrators.
//
// @Summary      Update a charge
// @Description  Update an existing charge by ID
// @Tags         Charges
// @Accept       json
// @Produce      json
// @Param        charge_id  path      string       true  "Charge ID"
// @Param        charge     body      dtos.Charge  true  "Charge Details"
// @Success      200        {object}  dtos.Charge
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/charges/{charge_id} [patch]
func UpdateChargeHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Charges", "charges.update"); !ok {
		return
	}

	id := c.Param("charge_id")

	req, ok := DecodeRequestBody[dtos.Charge](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Charges") {
		return
	}
	charge, err := models.UpdateCharge(models.DB, id, *req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Charges",
				Description: "Failed to update charge with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  c.Request,
			RawBody:  requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: chargeWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload: charge, Message: "Charge updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
		RawBody:  requestSummary,
	})
}

// Get Charge by ID
// GetChargeByIDHandler retrieves a charge by ID.
//
// @Summary      Get a charge by ID
// @Description  Retrieve a charge using its unique ID
// @Tags         Charges
// @Produce      json
// @Param        charge_id  path      string  true  "Charge ID"
// @Success      200        {object}  dtos.Charge
// @Failure      500        {object}  dtos.ErrorResponse
// @Router       /api/charges/{charge_id} [get]
func GetChargeByIDHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	id := c.Param("charge_id")
	charge, err := models.GetChargeByID(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: "Failed to get charge with ID " + id,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Charges",
		Description: chargeWithID + id + " retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: charge, Message: "Charge retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// Get All Charges
// GetAllChargesHandler retrieves all charges.
//
// @Summary      Get all charges
// @Description  Retrieve a list of all charges
// @Tags         Charges
// @Produce      json
// @Success      200  {object}  []dtos.Charge
// @Failure      500  {object}  dtos.ErrorResponse
// @Router       /api/charges [get]
func GetAllChargesHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	charges, err := models.GetAllCharges(models.DB)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: "Failed to get all charges",
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Charges",
		Description: "All charges retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: charges, Message: "Charges retrieved successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  c.Request,
		RawBody:  requestSummary})
}

// Delete Charge
// DeleteChargeHandler deletes a charge.
// This endpoint is restricted to administrators.
//
// @Summary      Delete a charge
// @Description  Delete a charge by ID
// @Tags         Charges
// @Produce      json
// @Param        charge_id  path      string  true  "Charge ID"
// @Success      200        {object}  map[string]any
// @Failure      500        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/charges/{charge_id} [delete]
func DeleteChargeHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Charges", "charges.delete"); !ok {
		return
	}

	id := c.Param("charge_id")
	if err := models.DeleteCharge(models.DB, id); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: "Failed to delete charge with ID " + id,
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Charges",
		Description: chargeWithID + id + " deleted successfully",
		Code:        http.StatusOK,
	},
		Payload:   nil,
		Message:   "Charge deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// Add charge to product
// AddChargeToProductHandler associates a charge with a product.
// This endpoint is restricted to administrators.
//
// @Summary      Add charge to product
// @Description  Associate a charge with a specific product
// @Tags         Charges
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.AddChargeToProductRequest  true  "Association Details"
// @Success      200      {object}  map[string]any
// @Failure      400      {object}  dtos.ErrorResponse
// @Failure      500      {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/charges [post]
func AddChargeToProductHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Charges", "products.create"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.AddChargeToProductRequest](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Charges") {
		return
	}
	err := models.AddChargeToProduct(models.DB, *req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: "Failed to add charge to product",
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Charges",
		Description: "Charge with charge ID " + req.ChargeID + " added to product with product ID " + req.ProductID + " successfully",
		Code:        http.StatusOK,
	},
		Payload:   nil,
		Message:   "Charge added to product successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
