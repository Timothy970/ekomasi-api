package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/services"
	"ekomasi_backend/utils"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// BulkUploadProductsHandler handles the bulk upload and validation of products via CSV file.
// Supports query params:
// - validate_only: 'true' to run a dry-run validation without DB writes
// - mode: 'upsert' (default), 'create_only', 'update_stock'
//
// @Summary      Bulk upload and validate products
// @Description  Upload multiple products via CSV with dry-run pre-validation and row-by-row error diagnostics
// @Tags         Products
// @Accept       multipart/form-data
// @Produce      json
// @Param        file           formData  file    true  "CSV File"
// @Param        validate_only  query     bool    false "If true, run dry-run pre-validation without DB writes"
// @Param        mode           query     string  false "Import mode: 'upsert', 'create_only', 'update_stock'"
// @Success      200   {object}  dtos.BulkImportDiagnosticResult
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      401   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/bulk-upload [post]
func BulkUploadProductsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}

	validateOnlyStr := c.Query("validate_only")
	validateOnly, _ := strconv.ParseBool(validateOnlyStr)
	mode := c.Query("mode")
	if mode == "" {
		mode = "upsert"
	}

	err := c.Request.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error parsing form: " + err.Error(),
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

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Csv file error: " + err.Error(),
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
	defer file.Close()

	products, err := utils.ParseProductsCSV(file)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Invalid CSV format: " + err.Error(),
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

	tenantID := fmt.Sprintf("%d", middleware.TenantIDFromContext(c.Request.Context()))
	diagnostic, err := services.ValidateAndProcessBulkImport(models.DB, tenantID, products, mode, validateOnly, authuser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Bulk import execution failed: " + err.Error(),
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

	msg := "Bulk product import processed successfully"
	if validateOnly {
		msg = "Bulk product pre-validation complete (Dry-Run)"
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: msg,
			Code:        http.StatusOK,
		},
		Payload:   diagnostic,
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// DownloadSampleCSVHandler generates and serves a sample CSV file for bulk product uploads.
//
// @Summary      Download sample CSV
// @Description  Download a sample CSV file template for bulk product upload
// @Tags         Products
// @Produce      text/csv
// @Success      200  {file}  file
// @Failure      500  {string} string "Error generating sample CSV"
// @Router       /api/products/sample-csv [get]
func DownloadSampleCSVHandler(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=sample_products.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	headers := utils.BulkUploadHeaders
	var headerNames []string
	for _, header := range headers {
		headerNames = append(headerNames, header.Name)
	}
	writer.Write(headerNames)

	// Sample products: 1 simple product & 1 parent-child variant combination product
	sampleData := [][]string{
		{
			"Organic Soap",
			"Natural handmade soap with essential oils",
			"SOAP001",
			"3.50",
			"Beauty",
			"100",
			"",
			"5",
			"123355677",
			"2.00",
			"12",
			"5",
			"3x3x1",
			"Nature's Best",
			"Nature's Best Co.",
			"12",
			"",
			"",
			"0.00",
			"",
			"",
			"",
		},
		{
			"Cotton Crewneck Tee",
			"Premium heavyweight organic cotton t-shirt",
			"TEE001",
			"25.00",
			"Fashion",
			"50",
			"apparel",
			"5",
			"987654321",
			"12.50",
			"0.35",
			"1",
			"10x12x1",
			"UrbanWear",
			"Urban Apparel Ltd",
			"12",
			"Blue / Medium",
			"TEE001-BL-M",
			"0.00",
			"Color: Blue",
			"Size: Medium",
			"",
		},
		{
			"Cotton Crewneck Tee",
			"Premium heavyweight organic cotton t-shirt",
			"TEE001",
			"25.00",
			"Fashion",
			"30",
			"apparel",
			"5",
			"987654321",
			"12.50",
			"0.35",
			"1",
			"10x12x1",
			"UrbanWear",
			"Urban Apparel Ltd",
			"12",
			"Blue / XL",
			"TEE001-BL-XL",
			"5.00",
			"Color: Blue",
			"Size: XL",
			"",
		},
	}

	for _, record := range sampleData {
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		c.String(http.StatusInternalServerError, "Error generating sample CSV")
		return
	}

	c.Header("X-Generated-At", time.Now().Format(time.RFC3339))
}

// validateUniqueSKUs checks if all SKUs in the products slice are unique
func validateUniqueSKUs(products []dtos.BulkUploadProduct) error {
	skuSet := make(map[string]bool)
	for _, product := range products {
		if _, exists := skuSet[product.SKU]; exists {
			return fmt.Errorf("duplicate SKU found in CSV: %s", product.SKU)
		}
		skuSet[product.SKU] = true
	}
	return nil
}
