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

const (
	inventoryView = "inventory.view"
	dateLayout    = "2006-01-02"
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
// @Success      200        {object}  map[string]any
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
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Inventory", inventoryView); !ok {
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
// @Success      200           {object}  map[string]any
// @Failure      400           {object}  dtos.ErrorResponse
// @Failure      409           {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/inventories/{inventory_id} [patch]
