package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func EditVoucherDesign(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.update"); !ok {
		return
	}
	designID := c.Param("voucher_id")

	// Parse multipart form (20 MB max)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
		})
		return
	}

	// Get image file (optional)
	var url string
	file, header, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()
		// Upload image to GCS
		url, err = utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return
		}
	}
	// Insert category into DB
	err = models.EditVoucherDesign(models.DB, designID, &url, c.Request.FormValue("name"), c.Request.FormValue("status"))
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher design",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher design updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func DeleteVoucherDesign(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", "promotions.delete"); !ok {
		return
	}
	designID := c.Param("voucher_id")

	err := models.DeleteVoucherDesign(models.DB, designID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to delete voucher design",
				Code:        http.StatusBadRequest,
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
			Module:      "Vouchers",
			Description: "Voucher design deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher design deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func GetAllVoucherDesigns(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	status := c.Query("status")
	q := c.Query("name")

	designs, pagination, err := models.GetAllVoucherDesigns(models.DB, page, limit, q, status)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher designs",
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
			Module:      "Vouchers",
			Description: "Voucher designs fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"designs": designs, "pagination": pagination},
		Message:   "Voucher designs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func GetVoucherDesignByID(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	designID := c.Param("voucher_id")

	design, err := models.GetVoucherDesign(models.DB, designID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher design",
				Code:        http.StatusNotFound,
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
			Module:      "Vouchers",
			Description: "Voucher design fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   design,
		Message:   "Voucher design fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func ListVoucherPurchasesHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", ""); !ok {
		return
	}

	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	q := c.Query("name")

	purchases, pagination, err := models.ListVoucherPurchases(models.DB, page, limit, q)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher purchases",
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
			Module:      "Vouchers",
			Description: "Voucher purchases fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"purchases": purchases, "pagination": pagination},
		Message:   "Voucher purchases fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func GetVoucherPurchasesHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Vouchers", ""); !ok {
		return
	}

	purchaseID := c.Param("voucher_id")

	purchase, err := models.GetVoucherPurchases(models.DB, purchaseID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher purchase",
				Code:        http.StatusNotFound,
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
			Module:      "Vouchers",
			Description: "Voucher purchase fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   purchase,
		Message:   "Voucher purchase fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
