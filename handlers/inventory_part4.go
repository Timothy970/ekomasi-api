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
