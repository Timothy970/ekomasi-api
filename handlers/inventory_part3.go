package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// @Param        end         query     string  false  "End Date (YYYY-MM-DD)"
// @Success      200         {object}  dtos.InventoryTurnoverResponse
// @Failure      404         {object}  dtos.ErrorResponse
// @Router       /api/reports/inventory/turnover/{product_id} [get]
func GetInventoryTurnoverByProduct(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	productID := c.Param("product_id")
	groupBy := "weekly"
	periodStr := c.Query("period")
	if periodStr != "" {
		groupBy = periodStr
	}
	startDate, endDate, _ := ParseDateRange(c.Request)

	data, err := models.GetInventoryTurnoverByProduct(models.DB, productID, startDate, endDate, groupBy)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get inventory turnover report for product " + productID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.InventoryTurnoverResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
			Type  string    `json:"type"`
		}{Start: startDate, End: endDate, Type: groupBy},
		Data: data,
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Product inventory turnover report generated successfully for product " + productID,
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Product inventory turnover report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// StockEntry manages inventory stock entry.
// It handles batch details, inspection details, inventory updates, and handling notes.
// This endpoint is restricted to administrators.
//
// @Summary      Manage inventory stock entry
// @Description  Create a new stock entry with batch and inspection details
// @Tags         Inventory
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id           formData  string  true   "Product ID"
// @Param        batch_number         formData  string  true   "Batch Number"
// @Param        expiry_date          formData  string  true   "Expiry Date (YYYY-MM-DD)"
// @Param        manufacturing_date   formData  string  true   "Manufacturing Date (YYYY-MM-DD)"
// @Param        inspection_date      formData  string  true   "Inspection Date (YYYY-MM-DD)"
// @Param        inspector_id         formData  string  true   "Inspector ID"
// @Param        inspection_notes     formData  string  false  "Inspection Notes"
// @Param        quantity_received    formData  int     true   "Quantity Received"
// @Param        minimum_stock_level  formData  int     true   "Minimum Stock Level"
// @Param        store_quantity       formData  string  true   "Store Quantity JSON"
// @Param        supplier_id          formData  string  true   "Supplier ID"
// @Param        purchase_order_id    formData  string  true   "Purchase Order ID"
// @Param        buying_price         formData  number  true   "Buying Price"
// @Param        condition_id         formData  string  true   "Condition ID"
// @Param        handling_notes       formData  string  false  "Handling Notes"
// @Param        batch_images         formData  file    false  "Batch Images"
// @Param        inspection_images    formData  file    false  "Inspection Images"
// @Success      201                  {object}  map[string]interface{}
// @Failure      400                  {object}  dtos.ErrorResponse
// @Failure      404                  {object}  dtos.ErrorResponse
// @Failure      500                  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/inventory/stock-entry [post]
func StockEntry(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Check if user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.create"); !ok {
		return
	}

	// Parse multipart form (100 MB limit for high-quality images)
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to parse multipart form when creating stock entry",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Upload images
	batchImageUrls, err := handleImageUpload(c.Request, "batch_images")
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to upload batch images when creating stock entry",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	inspectionImageUrls, err := handleImageUpload(c.Request, "inspection_images")
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to upload inspection images when creating stock entry",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Build stock entry request
	var supplierID *string
	if supplierIDValue := c.Request.FormValue("supplier_id"); supplierIDValue != "" {
		supplierID = &supplierIDValue
	}
	inspectionNotes := c.Request.FormValue("inspection_notes")
	handlingNotes := c.Request.FormValue("handling_notes")
	req := &dtos.StockEntryRequest{
		ProductID:         c.Request.FormValue("product_id"),
		BatchImages:       &batchImageUrls,
		BatchNumber:       c.Request.FormValue("batch_number"),
		ExpiryDate:        c.Request.FormValue("expiry_date"),
		ManufacturingDate: c.Request.FormValue("manufacturing_date"),
		InspectionDate:    c.Request.FormValue("inspection_date"),
		InspectionImage:   &inspectionImageUrls,
		InspectorID:       c.Request.FormValue("inspector_id"),
		InspectionNotes:   &inspectionNotes,
		QuantityReceived:  parseInt(c.Request.FormValue("quantity_received")),
		MinimumStockLevel: parseInt(c.Request.FormValue("minimum_stock_level")),
		StoreQuantity:     ParseStoreInfoArray(c.Request.FormValue("store_quantity")),
		SupplierID:        supplierID,
		BuyingPrice:       parseFloat(c.Request.FormValue("buying_price")),
		HandlingNotes:     &handlingNotes,
		SellingPrice:      parseFloat(c.Request.FormValue("selling_price")),
		VariantQuantity:   ParseVariantQuantityArray(c.Request.FormValue("variant_quantity")),
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Inventory") {
		return
	}
	// Parse dates for comparison
	expiryDate, err := time.Parse("2006-01-02", req.ExpiryDate)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Invalid expiry date format when creating stock entry",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid expiry date format. Expected YYYY-MM-DD",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	mfgDate, err := time.Parse("2006-01-02", req.ManufacturingDate)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Invalid manufacturing date format when creating stock entry",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid manufacturing date format. Expected YYYY-MM-DD",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	inspectionDate, err := time.Parse("2006-01-02", req.InspectionDate)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Invalid inspection date format when creating stock entry",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid inspection date format. Expected YYYY-MM-DD",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//inspection date cannot be in the future
	if inspectionDate.After(time.Now()) {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Inspection date cannot be in the future when creating stock entry",
				Code:        http.StatusBadRequest,
			},
			Message:   "Inspection date cannot be in the future",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	if expiryDate.Before(mfgDate) || expiryDate.Equal(mfgDate) {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Expiry date must be after manufacturing date when creating stock entry",
				Code:        http.StatusBadRequest,
			},
			Message:   "Expiry date must be after manufacturing date",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Validate that store quantities sum equals quantity received
	totalStoreQuantity := 0
	for _, store := range req.StoreQuantity {
		totalStoreQuantity += store.Quantity
	}
	if totalStoreQuantity != req.QuantityReceived {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: fmt.Sprintf("Store quantities sum (%d) does not match quantity received (%d)", totalStoreQuantity, req.QuantityReceived),
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("The sum of store quantities (%d) must equal the total quantity received (%d)", totalStoreQuantity, req.QuantityReceived),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Start Transaction
	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to start transaction when creating stock entry",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Internal Server Error",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Single defer with proper cleanup
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-panic after rollback
		}
	}()

	var invetoryIDS []string
	for _, warehouse := range req.StoreQuantity {
		storeID := warehouse.StoreID
		quantity := warehouse.Quantity
		inventoryID, err := handleInventoryTracking(tx, req, storeID, quantity)
		if err != nil {
			tx.Rollback()
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to store inventory tracking when creating stock entry",
					Code:        http.StatusNotFound,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		invetoryIDS = append(invetoryIDS, inventoryID)
		batchID, err := handleBatch(tx, req, inventoryID)
		if err != nil {
			tx.Rollback()
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to store batch details when creating stock entry",
					Code:        http.StatusNotFound,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		err = handleInspection(tx, req, batchID)
		if err != nil {
			tx.Rollback()
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to store inspection details when creating stock entry",
					Code:        http.StatusNotFound,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
		err = handleStoreConditonsAndNotes(tx, req, batchID)
		if err != nil {
			tx.Rollback()
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to store handling notes when creating stock entry",
					Code:        http.StatusNotFound,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}
	//update product buying price and selling price
	if err := models.UpdateProductPrices(tx, req.ProductID, req.BuyingPrice, req.SellingPrice); err != nil {
		tx.Rollback()
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to update product prices when creating stock entry",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// update variant quantities if applicable
	if len(req.VariantQuantity) > 0 {
		if err := models.UpdateVariantQuantities(tx, req.VariantQuantity); err != nil {
			tx.Rollback()
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to update variant quantities when creating stock entry",
					Code:        http.StatusNotFound,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}
	// Commit Transaction
	if err := tx.Commit(); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to commit transaction when creating stock entry",
				Code:        http.StatusInternalServerError,
			},
			Message:   "Internal Server Error",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory(stock entry) created successfully with inventory IDs " + fmt.Sprint(invetoryIDS),
			Code:        http.StatusCreated,
		},
		Payload:   map[string]any{"inventory_ids": invetoryIDS},
		Message:   "Inventory created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
