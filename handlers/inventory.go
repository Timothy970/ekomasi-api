package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// List all inventories
//
// @Summary List all inventories
// @Description List all inventories
// @Tags Inventories
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/inventories [get]
func ListInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}
	categoryID := r.URL.Query().Get("category_id")
	stock := r.URL.Query().Get("stock")
	storeID := r.URL.Query().Get("store_id")
	useCached := categoryID == "" && stock == "" && storeID == ""
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyInventories := fmt.Sprintf("inventories_%d_size_%d", page, size)
	var inventories []dtos.Inventory
	var cachedInventories []dtos.Inventory
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	cacheKeyPagination := fmt.Sprintf("inventories_pagination_%d_size_%d", page, size)
	if useCached {

		_ = utils.GetCache(cacheKeyInventories, &cachedInventories)
		_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	}
	if cachedInventories == nil {
		var err error
		inventories, pagination, err = models.ListInventory(page, size, categoryID, stock, storeID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusBadRequest,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		if useCached {
			_ = utils.SetCache(cacheKeyInventories, cachedInventories)
			_ = utils.SetCache(cacheKeyPagination, cachedPagination)
		}
	} else {
		inventories = cachedInventories
		pagination = cachedPagination
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
		Payload: dtos.InventoryListResponse{
			Meta:        *pagination,
			Inventories: inventories,
		},
		Message:   "Inventories",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Add a new inventory
//
// @Summary Add a new inventory
// @Description Add a new inventory
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/inventories [post]
func CreateInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateInventoryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	if err := models.CreateInventory(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("inventories_")
	utils.DeleteCacheByPrefix("inventories_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Inventory created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get inventory by ID
//
// @Summary List inventory by ID
// @Description List inventory by ID
// @Tags Inventories
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/inventories/{inventory_id} [get]
func GetInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	inv, err := models.GetInventory(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   inv,
		Message:   "Inventory fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Update inventory
//
// @Summary Update inventory by ID
// @Description Update inventory by ID
// @Tags Inventories
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/inventories/{inventory_id} [patch]
func UpdateInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateInventoryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if req.LowStockThreshold == nil && req.Quantity == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   "Request cannot be empty",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	if err := models.UpdateInventory(id, req.Quantity, req.LowStockThreshold); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("inventories_")
	utils.DeleteCacheByPrefix("inventories_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Inventory updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete an inventory
//
// @Summary Delete inventory by ID
// @Description Delete inventory by ID
// @Tags Inventories
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/inventories/{inventory_id} [delete]
func DeleteInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	if err := models.DeleteInventory(id); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("inventories_")
	utils.DeleteCacheByPrefix("inventories_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Inventory deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetInventoryTurnover(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	groupBy := "weekly"
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		groupBy = periodStr
	}
	start, end, _ := ParseDateRange(r)
	data, err := models.GetInventoryTurnover(start, end, groupBy)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.InventoryTurnoverResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
			Type  string    `json:"type"`
		}{Start: start, End: end, Type: groupBy},
		Data: data,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   resp,
		Message:   "Summary inventory turnover report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetInventoryTurnoverByProduct(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	productID := mux.Vars(r)["product_id"]
	groupBy := "weekly"
	periodStr := r.URL.Query().Get("period")
	if periodStr != "" {
		groupBy = periodStr
	}
	start, end, _ := ParseDateRange(r)

	data, err := models.GetInventoryTurnoverByProduct(productID, start, end, groupBy)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.InventoryTurnoverResponse{
		Period: struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
			Type  string    `json:"type"`
		}{Start: start, End: end, Type: groupBy},
		Data: data,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   resp,
		Message:   "Product inventory turnover report",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Manage inventory stock entry
// Insert Batch details
// Insert Inspection details
// Update inventory
// Store handling notes
func StockEntry(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Check if user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	// Parse multipart form (20 MB limit)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Upload images
	batchImageUrls, err := handleImageUpload(r, "batch_images")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	inspectionImageUrls, err := handleImageUpload(r, "inspection_images")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusNotFound,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Build stock entry request
	req := &dtos.StockEntryRequest{
		ProductID:         r.FormValue("product_id"),
		BatchImages:       &batchImageUrls,
		BatchNumber:       r.FormValue("batch_number"),
		ExpiryDate:        r.FormValue("expiry_date"),
		ManufacturingDate: r.FormValue("manufacturing_date"),
		InspectionDate:    r.FormValue("inspection_date"),
		InspectionImage:   &inspectionImageUrls,
		InspectorID:       r.FormValue("inspector_id"),
		InspectionNotes:   r.FormValue("inspection_notes"),
		QuantityReceived:  parseInt(r.FormValue("quantity_received")),
		MinimumStockLevel: parseInt(r.FormValue("minimum_stock_level")),
		StoreQuantity:     ParseStoreInfoArray(r.FormValue("store_quantity")),
		SupplierID:        r.FormValue("supplier_id"),
		PurchaseOrderID:   r.FormValue("purchase_order_id"),
		BuyingPrice:       parseFloat(r.FormValue("buying_price")),
		ConditionID:       r.FormValue("condition_id"),
		HandlingNotes:     r.FormValue("handling_notes"),
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	var invetoryIDS []string
	for _, warehouse := range req.StoreQuantity {
		storeID := warehouse.StoreID
		inventoryID, err := handleInventoryTracking(req, storeID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		invetoryIDS = append(invetoryIDS, inventoryID)
		batchID, err := handleBatch(req, inventoryID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		err = handleInspection(req, batchID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		err = handleStoreConditonsAndNotes(req, batchID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   map[string]any{"inventory_ids": invetoryIDS},
		Message:   "Inventory created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func handleImageUpload(r *http.Request, field string) ([]string, error) {
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
func handleInventoryTracking(req *dtos.StockEntryRequest, storeID string) (string, error) {

	inventoryData := dtos.InventoryTracking{
		ProductID:         req.ProductID,
		Quantity:          req.QuantityReceived,
		LowStockThreshold: req.MinimumStockLevel,
		StoreID:           storeID}
	// insert into db
	inventoryID, err := models.StoreInventoryTracking(inventoryData)
	if err != nil {
		return "", err
	}
	return inventoryID, nil
}

func handleBatch(req *dtos.StockEntryRequest, inventoryID string) (string, error) {
	//  Store batch details in DB along with urls
	batchData := dtos.Batch{
		InventoryID:       inventoryID,
		BatchNumber:       req.BatchNumber,
		Images:            *req.BatchImages,
		ExpiryDate:        req.ExpiryDate,
		ManufacturingDate: req.ManufacturingDate,
	}
	batchID, err := models.StoreBatchDetails(batchData)
	if err != nil {
		return "", err
	}

	return batchID, nil
}
func handleInspection(req *dtos.StockEntryRequest, batchID string) error {

	// Store inspection details in DB
	inspectionData := dtos.Inspection{
		BatchID:         batchID,
		InspectionDate:  req.InspectionDate,
		InspectorID:     req.InspectorID,
		InspectionNotes: req.InspectionNotes,
		Images:          *req.InspectionImage,
	}
	err := models.StoreInspectionDetails(inspectionData)
	if err != nil {
		return err
	}

	return nil
}

func handleStoreConditonsAndNotes(req *dtos.StockEntryRequest, batchID string) error {
	conditionData := dtos.InventoryCondition{
		BatchID:       batchID,
		ConditionID:   req.ConditionID,
		HandlingNotes: req.HandlingNotes,
	}
	// Update inventory quantity
	err := models.StoreHandlingNotes(conditionData)
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
