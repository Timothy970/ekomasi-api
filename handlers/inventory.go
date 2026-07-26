package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ListInventory retrieves a paginated list of inventories.
// This endpoint is restricted to administrators.
//
// @Summary      List all inventories
// @Description  List all inventories
// @Tags         Inventories
// @Produce      json
// @Param        page         query     int     false  "Page number"
// @Param        size         query     int     false  "Page size"
// @Param        category_id  query     string  false  "Category ID"
// @Param        stock        query     string  false  "Stock Status"
// @Param        store_id     query     string  false  "Store ID"
// @Param        q            query     string  false  "Search Query"
// @Success      200          {object}  dtos.InventoryListResponse
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/inventories [get]
func ListInventory(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", ""); !ok {
		return
	}
	categoryID := c.Query("category_id")
	stock := c.Query("stock")
	storeID := c.Query("store_id")
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	q := c.Query("q")
	inventories, pagination, err := models.ListInventory(models.DB, page, size, categoryID, stock, storeID, q)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to list inventory",
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
			Module:      "Inventory",
			Description: "Inventories fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: dtos.InventoryListResponse{
			Meta:        *pagination,
			Inventories: inventories,
		},
		Message:   "Inventories",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// CreateInventory creates a new inventory item.
// This endpoint is restricted to administrators.
//
// @Summary      Add a new inventory
// @Description  Add a new inventory
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        inventory  body      dtos.CreateInventoryRequest  true  "Inventory Details"
// @Success      200        {object}  map[string]interface{}
// @Failure      400        {object}  dtos.ErrorResponse
// @Failure      409        {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/inventories [post]
func CreateInventory(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.create")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateInventoryRequest](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Inventory") {
		return
	}

	if err := models.CreateInventory(models.DB, *req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to create inventory",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("inventories_")
	utils.DeleteCacheByPrefix("inventories_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory saved successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Inventory created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetInventory retrieves an inventory item by ID.
// This endpoint is restricted to administrators.
//
// @Summary      List inventory by ID
// @Description  List inventory by ID
// @Tags         Inventories
// @Produce      json
// @Param        inventory_id  path      string  true  "Inventory ID"
// @Success      200           {object}  dtos.Inventory
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      409           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/inventories/{inventory_id} [get]
func GetInventory(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", ""); !ok {
		return
	}
	id := c.Param("inventory_id")

	inv, err := models.GetInventory(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory : " + err.Error(),
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
			Module:      "Inventory",
			Description: "Inventory fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   inv,
		Message:   "Inventory fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DownloadInventoryCSV downloads inventory data as a CSV file.
// This endpoint is restricted to administrators.
//
// @Summary      Download inventory CSV
// @Description  Download inventory data as CSV
// @Tags         Inventories
// @Produce      text/csv
// @Param        inventory_id  path      string  true  "Inventory ID"
// @Success      200           {file}    file
// @Failure      404           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/inventories/{inventory_id}/csv [get]
func DownloadInventoryCSV(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", ""); !ok {
		return
	}
	id := c.Param("inventory_id")

	inv, err := models.GetInventory(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=inventory.csv")

	if err := utils.ExportInventoryCSV(c.Writer, *inv); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to export inventory CSV",
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

// DownloadInventoryPDF downloads inventory data as a PDF file.
// This endpoint is restricted to administrators.
//
// @Summary      Download inventory PDF
// @Description  Download inventory data as PDF
// @Tags         Inventories
// @Produce      application/pdf
// @Param        inventory_id  path      string  true  "Inventory ID"
// @Success      200           {file}    file
// @Failure      404           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/inventories/{inventory_id}/pdf [get]
func DownloadInventoryPDF(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.view"); !ok {
		return
	}
	id := c.Param("inventory_id")

	inv, err := models.GetInventory(models.DB, id)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	pdfBytes, err := utils.GenerateInventoryPDF(*inv)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to generate inventory PDF",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return

	}

	filename := fmt.Sprintf("inventory_%s.pdf", inv.InventoryID)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Writer.Write(pdfBytes)
}

// UpdateInventory updates an existing inventory item.
// This endpoint is restricted to administrators.
//
// @Summary      Update inventory by ID
// @Description  Update inventory by ID
// @Tags         Inventories
// @Accept       json
// @Produce      json
// @Param        inventory_id  path      string                      true  "Inventory ID"
// @Param        inventory     body      dtos.UpdateInventoryRequest true  "Inventory Details"
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      409           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/inventories/{inventory_id} [patch]
func UpdateInventory(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.update")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateInventoryRequest](c, requestSummary, start)
	if !ok {
		return
	}
	if req.LowStockThreshold == nil && req.Quantity == nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "LowStockThreshold or Quantity must be provided",
				Code:        http.StatusBadRequest,
			},
			Message:   "Request cannot be empty",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Inventory") {
		return
	}
	id := c.Param("inventory_id")

	if err := models.UpdateInventory(models.DB, id, req.Quantity, req.LowStockThreshold); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to update inventory with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("inventories_")
	utils.DeleteCacheByPrefix("inventories_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Inventory updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteInventory deletes an inventory item.
// This endpoint is restricted to administrators.
//
// @Summary      Delete inventory by ID
// @Description  Delete inventory by ID
// @Tags         Inventories
// @Produce      json
// @Param        inventory_id  path      string  true  "Inventory ID"
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      409           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/inventories/{inventory_id} [delete]
func DeleteInventory(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.delete")
	if !ok {
		return
	}
	id := c.Param("inventory_id")

	if err := models.DeleteInventory(models.DB, id); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to delete inventory with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("inventories_")
	utils.DeleteCacheByPrefix("inventories_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Inventory deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetInventoryTurnover retrieves inventory turnover report.
//
// @Summary      Get inventory turnover report
// @Description  Get inventory turnover report
// @Tags         Reports
// @Produce      json
// @Param        period  query     string  false  "Period (weekly, monthly, etc.)"
// @Param        start   query     string  false  "Start Date (YYYY-MM-DD)"
// @Param        end     query     string  false  "End Date (YYYY-MM-DD)"
// @Success      200     {object}  dtos.InventoryTurnoverResponse
// @Failure      404     {object}  dtos.ErrorResponse
// @Router       /api/reports/inventory/turnover [get]
func GetInventoryTurnover(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	groupBy := "weekly"
	periodStr := c.Query("period")
	if periodStr != "" {
		groupBy = periodStr
	}
	startDate, endDate, _ := ParseDateRange(c.Request)
	data, err := models.GetInventoryTurnover(models.DB, startDate, endDate, groupBy)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get inventory turnover report",
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
			Description: "Summary inventory turnover report generated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   resp,
		Message:   "Summary inventory turnover report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetInventoryTurnoverByProduct retrieves inventory turnover report for a specific product.
//
// @Summary      Get inventory turnover report by product
// @Description  Get inventory turnover report by product
// @Tags         Reports
// @Produce      json
// @Param        product_id  path      string  true   "Product ID"
// @Param        period      query     string  false  "Period (weekly, monthly, etc.)"
// @Param        start       query     string  false  "Start Date (YYYY-MM-DD)"
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

func handleImageUpload(r *http.Request, field string) ([]string, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	files := r.MultipartForm.File[field]
	if len(files) == 0 {
		return nil, nil
	}

	var urls []string
	for _, fh := range files {
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{fh})
		if err != nil {
			return nil, fmt.Errorf("failed to upload %s: %w", field, err)
		}
		urls = append(urls, url)
	}
	return urls, nil
}
func handleInventoryTracking(db models.DBExecutor, req *dtos.StockEntryRequest, storeID string, quantity int) (string, error) {

	inventoryData := dtos.InventoryTracking{
		ProductID:         req.ProductID,
		Quantity:          quantity,
		LowStockThreshold: req.MinimumStockLevel,
		StoreID:           storeID,
		SupplierID:        req.SupplierID}
	// insert into db
	inventoryID, err := models.StoreInventoryTracking(db, inventoryData)
	if err != nil {
		return "", err
	}
	return inventoryID, nil
}

func handleBatch(db models.DBExecutor, req *dtos.StockEntryRequest, inventoryID string) (string, error) {
	//  Store batch details in DB along with urls
	var images []string
	if req.BatchImages != nil {
		images = *req.BatchImages
	}

	batchData := dtos.Batch{
		InventoryID:       inventoryID,
		BatchNumber:       req.BatchNumber,
		Images:            images,
		ExpiryDate:        req.ExpiryDate,
		ManufacturingDate: req.ManufacturingDate,
	}
	batchID, err := models.StoreBatchDetails(db, batchData)
	if err != nil {
		return "", err
	}

	return batchID, nil
}
func handleInspection(db models.DBExecutor, req *dtos.StockEntryRequest, batchID string) error {

	// Store inspection details in DB
	var images []string
	if req.InspectionImage != nil {
		images = *req.InspectionImage
	}

	inspectionData := dtos.Inspection{
		BatchID:         batchID,
		InspectionDate:  req.InspectionDate,
		InspectorID:     req.InspectorID,
		InspectionNotes: req.InspectionNotes,
		Images:          images,
	}
	err := models.StoreInspectionDetails(db, inspectionData)
	if err != nil {
		return err
	}

	return nil
}

func handleStoreConditonsAndNotes(db models.DBExecutor, req *dtos.StockEntryRequest, batchID string) error {
	conditionData := dtos.InventoryCondition{
		BatchID:       batchID,
		HandlingNotes: req.HandlingNotes,
	}
	// Update inventory quantity
	err := models.StoreHandlingNotes(db, conditionData)
	if err != nil {
		return err
	}
	return nil
}

// parseInt safely parses a string to int, returns 0 if parsing fails
func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// parseFloat safely parses a string to float64, returns 0 if parsing fails
func parseFloat(s string) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

// ParseStoreInfoArray parses a JSON array string into a slice of dtos.StoreInfo.
// Returns an empty slice if parsing fails.
func ParseStoreInfoArray(s string) []dtos.StoreInfo {

	var stores []dtos.StoreInfo
	if s == "" {
		return stores
	}
	err := json.Unmarshal([]byte(s), &stores)
	if err != nil {
		return stores
	}
	return stores
}
func ParseVariantQuantityArray(s string) []dtos.VariantQuantity {

	var variants []dtos.VariantQuantity
	if s == "" {
		return variants
	}
	err := json.Unmarshal([]byte(s), &variants)
	if err != nil {
		return variants
	}
	return variants
}

func GetInventoryStockSummary(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.view"); !ok {
		return
	}
	id := c.Param("inventory_id")
	//add filter by store id
	storeID := c.Query("store_id")
	inv, err := models.GetInventoryStockSummary(models.DB, id, storeID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory stock summary for inventory with ID " + id,
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
			Module:      "Inventory",
			Description: "Successfully fetched inventory stock summary for inventory with ID " + id,
			Code:        http.StatusOK,
		},
		Payload:   inv,
		Message:   "Inventory stock summary fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

func GetInventoryStockHistory(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", "inventory.view"); !ok {
		return
	}
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	id := c.Param("inventory_id")
	//add filter by store id
	inv, pagination, err := models.GetInventoryStockHistory(models.DB, id, page, size)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory stock history for inventory with ID " + id,
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
			Module:      "Inventory",
			Description: "Successfully fetched inventory stock history for inventory with ID " + id,
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"history": inv, "pagination": pagination},
		Message:   "Inventory stock history fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
