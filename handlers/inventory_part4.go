package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

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
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", inventoryView); !ok {
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
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", inventoryView); !ok {
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

func validateStockEntryDates(req *dtos.StockEntryRequest) (string, string) {
	expiryDate, err := time.Parse(dateLayout, req.ExpiryDate)
	if err != nil {
		return "Invalid expiry date format. Expected YYYY-MM-DD", "Invalid expiry date format when creating stock entry"
	}

	mfgDate, err := time.Parse(dateLayout, req.ManufacturingDate)
	if err != nil {
		return "Invalid manufacturing date format. Expected YYYY-MM-DD", "Invalid manufacturing date format when creating stock entry"
	}

	inspectionDate, err := time.Parse(dateLayout, req.InspectionDate)
	if err != nil {
		return "Invalid inspection date format. Expected YYYY-MM-DD", "Invalid inspection date format when creating stock entry"
	}

	if inspectionDate.After(time.Now()) {
		return "Inspection date cannot be in the future", "Inspection date cannot be in the future when creating stock entry"
	}

	if expiryDate.Before(mfgDate) || expiryDate.Equal(mfgDate) {
		return "Expiry date must be after manufacturing date", "Expiry date must be after manufacturing date when creating stock entry"
	}

	return "", ""
}

func processStoreQuantities(tx models.DBExecutor, req *dtos.StockEntryRequest) ([]string, string, error) {
	var inventoryIDs []string
	for _, warehouse := range req.StoreQuantity {
		storeID := warehouse.StoreID
		quantity := warehouse.Quantity
		inventoryID, err := handleInventoryTracking(tx, req, storeID, quantity)
		if err != nil {
			return nil, "Failed to store inventory tracking when creating stock entry", err
		}
		inventoryIDs = append(inventoryIDs, inventoryID)
		batchID, err := handleBatch(tx, req, inventoryID)
		if err != nil {
			return nil, "Failed to store batch details when creating stock entry", err
		}
		if err := handleInspection(tx, req, batchID); err != nil {
			return nil, "Failed to store inspection details when creating stock entry", err
		}
		if err := handleStoreConditonsAndNotes(tx, req, batchID); err != nil {
			return nil, "Failed to store handling notes when creating stock entry", err
		}
	}
	return inventoryIDs, "", nil
}
