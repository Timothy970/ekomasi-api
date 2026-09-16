package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
	// Parse and validate dates
	if msg, desc := validateStockEntryDates(req); msg != "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: desc,
				Code:        http.StatusBadRequest,
			},
			Message:   msg,
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

	invetoryIDS, errDesc, err := processStoreQuantities(tx, req)
	if err != nil {
		tx.Rollback()
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Inventory",
				Description: errDesc,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
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
