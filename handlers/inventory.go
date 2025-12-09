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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory"); !ok {
		return
	}
	categoryID := r.URL.Query().Get("category_id")
	stock := r.URL.Query().Get("stock")
	storeID := r.URL.Query().Get("store_id")
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	q := r.URL.Query().Get("q")
	inventories, pagination, err := models.ListInventory(page, size, categoryID, stock, storeID, q)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to list inventory",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateInventoryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Inventory") {
		return
	}

	if err := models.CreateInventory(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to create inventory",
				Code:        http.StatusInternalServerError,
			},
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
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory saved successfully",
			Code:        http.StatusCreated,
		},
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory"); !ok {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	inv, err := models.GetInventory(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory : " + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   inv,
		Message:   "Inventory fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func DownloadInventoryCSV(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory"); !ok {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	inv, err := models.GetInventory(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=inventory.csv")

	if err := utils.ExportInventoryCSV(w, *inv); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to export inventory CSV",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return

	}
}

func DownloadInventoryPDF(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory"); !ok {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	inv, err := models.GetInventory(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	pdfBytes, err := utils.GenerateInventoryPDF(*inv)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to generate inventory PDF",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return

	}

	filename := fmt.Sprintf("inventory_%s.pdf", inv.InventoryID)

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(pdfBytes)
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
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateInventoryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if req.LowStockThreshold == nil && req.Quantity == nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "LowStockThreshold or Quantity must be provided",
				Code:        http.StatusBadRequest,
			},
			Message:   "Request cannot be empty",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Inventory") {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	if err := models.UpdateInventory(id, req.Quantity, req.LowStockThreshold); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to update inventory with ID " + id,
				Code:        http.StatusNotFound,
			},
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
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
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
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory")
	if !ok {
		return
	}
	id := mux.Vars(r)["inventory_id"]

	if err := models.DeleteInventory(id); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to delete inventory with ID " + id,
				Code:        http.StatusNotFound,
			},
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
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
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
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get inventory turnover report",
				Code:        http.StatusNotFound,
			},
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
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Summary inventory turnover report generated successfully",
			Code:        http.StatusCreated,
		},
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
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to get inventory turnover report for product " + productID,
				Code:        http.StatusNotFound,
			},
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
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Product inventory turnover report generated successfully for product " + productID,
			Code:        http.StatusCreated,
		},
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory"); !ok {
		return
	}

	// Parse multipart form (20 MB limit)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to parse multipart form when creating stock entry",
				Code:        http.StatusNotFound,
			},
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
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to upload batch images when creating stock entry",
				Code:        http.StatusNotFound,
			},
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
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to upload inspection images when creating stock entry",
				Code:        http.StatusNotFound,
			},
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
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Inventory") {
		return
	}
	var invetoryIDS []string
	for _, warehouse := range req.StoreQuantity {
		storeID := warehouse.StoreID
		inventoryID, err := handleInventoryTracking(req, storeID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to store inventory tracking when creating stock entry",
					Code:        http.StatusNotFound,
				},
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
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to store batch details when creating stock entry",
					Code:        http.StatusNotFound,
				},
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
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to store inspection details when creating stock entry",
					Code:        http.StatusNotFound,
				},
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
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Inventory",
					Description: "Failed to store handling notes when creating stock entry",
					Code:        http.StatusNotFound,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Inventory(stock entry) created successfully with inventory IDs " + fmt.Sprint(invetoryIDS),
			Code:        http.StatusCreated,
		},
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
		StoreID:           storeID,
		SupplierID:        req.SupplierID}
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

func GetInventoryStockSummary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory"); !ok {
		return
	}
	id := mux.Vars(r)["inventory_id"]
	//add filter by store id
	storeID := r.URL.Query().Get("store_id")
	inv, err := models.GetInventoryStockSummary(id, storeID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory stock summary for inventory with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Successfully fetched inventory stock summary for inventory with ID " + id,
			Code:        http.StatusOK,
		},
		Payload:   inv,
		Message:   "Inventory stock summary fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetInventoryStockHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Inventory"); !ok {
		return
	}
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	id := mux.Vars(r)["inventory_id"]
	//add filter by store id
	inv, pagination, err := models.GetInventoryStockHistory(id, page, size)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: "Failed to get inventory stock history for inventory with ID " + id,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Inventory",
			Description: "Successfully fetched inventory stock history for inventory with ID " + id,
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"history": inv, "pagination": pagination},
		Message:   "Inventory stock history fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
