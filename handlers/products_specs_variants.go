package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func HandleProductSpecificationsUpdate(c *gin.Context) {
	state := "update"
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	// Ensure the user is an admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.update")
	if !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.ProductSpecification](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	//handle products specifications
	err := handleProductSpecs(models.DB, *req, state)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product specifications: " + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//handle products variants
	err = handleProductsVariants(models.DB, *req, state)

	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product variants: " + err.Error(),
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//handle product warranty
	err = handleProductsWarranty(models.DB, *req)

	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product warranty: " + err.Error(),
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
			Module:      "Products",
			Description: "Product specifications updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product specifications updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

func handleProductSpecs(db models.DBExecutor, req dtos.ProductSpecification, state string) error {
	data := dtos.ProductSpecs{
		ProductID:    req.ProductID,
		Weight:       req.Weight,
		WeightLimit:  req.WeightLimit,
		Dimensions:   req.Dimensions,
		Manufacturer: req.Manufacturer,
	}
	//if updating first hold existing specs
	specs := []string{}
	var err error
	if state == "update" {
		specs, err = models.HoldProductSpecs(db, req.ProductID)
		if err != nil {
			return err
		}
	}
	err = models.InsertProductSpecs(db, data)
	if err != nil {
		return err
	}
	//if updating remove held specs
	if state == "update" {
		for _, specID := range specs {
			err := models.RemoveHeldProductSpecs(db, specID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
func handleProductsVariants(db models.DBExecutor, req dtos.ProductSpecification, state string) error {
	data := dtos.ProductVariantRequest{
		ProductID: req.ProductID,
	}
	noVariantMsg := "variant not found"

	// Map each variant type to its IDs
	variantGroups := map[string][]string{
		"brand":        toSlice(req.Brand),        // handle single value as slice
		"manufacturer": toSlice(req.Manufacturer), // handle single value as slice
	}
	// If updating, first hold existing variants
	var err error
	if state == "update" {
		err = models.HoldProductVariants(db, req.ProductID)
		if err != nil {
			return err
		}
		//also hold existing combinations if updating
		err = models.DeleteVariantSelectionsByProductID(db, req.ProductID)
		if err != nil {
			return err
		}
	}
	// Loop through each variant group
	for variantType, variantIDs := range variantGroups {
		for _, id := range variantIDs {
			if id == "" {
				continue
			}
			if err := addProductVariantWithHandling(db, id, variantType, data, noVariantMsg); err != nil {
				return err
			}
		}
	}
	// now handle VariantSelections.
	for _, variantSelection := range req.VariantSelections {
		// 1. Insert combination
		combinationID, err := models.InsertCombination(db, dtos.Combination{
			ProductID:       req.ProductID,
			Name:            variantSelection.Name,
			SKU:             variantSelection.SKU,
			AdditionalPrice: variantSelection.AdditionalPrice,
		})
		if err != nil {
			return err
		}

		// 2. Insert mapping (VERY IMPORTANT)
		for _, variantID := range variantSelection.VariantIDs {
			err := models.InsertCombinationOption(db, combinationID, variantID, req.ProductID)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func toSlice(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func addProductVariantWithHandling(db models.DBExecutor, variantID, variantType string, data dtos.ProductVariantRequest, notFoundMsg string) error {
	err := models.AddProductVariant(db, variantID, data)
	if err == nil {
		return nil
	}
	if err.Error() == notFoundMsg {
		log.Printf("%s variant with variant_id %s not found", variantType, variantID)
		return nil
	}
	return err
}

// handleProductsWarranty creates or updates warranty information for a product.
// It takes product specification data and creates a warranty record in the database.
func handleProductsWarranty(db models.DBExecutor, req dtos.ProductSpecification) error {
	// Build warranty data transfer object from product specification
	data := dtos.AddProductWarrantiesRequest{
		ProductID:      req.ProductID,      // Product identifier
		WarrantyTypeID: req.WarrantyType,   // Type of warranty (manufacturer, extended, etc.)
		WarrantyPeriod: req.WarrantyPeriod, // Duration of warranty coverage
	}

	// Insert warranty record into database
	err := models.AddProductWarranties(db, data)
	return err
}

// updateProductDiscount handles the atomic update of product promotions.
// It holds existing promotions, adds new ones, then removes held promotions.
func updateProductDiscount(productID string, data dtos.AddPromotionToProductRequest) error {
	// Temporarily hold current promotions
	promotionIDs, err := models.HoldProductPromotions(models.DB, productID)
	if err != nil {
		return err
	}

	// Add new promotion to product
	err = models.AddPromotionToProduct(models.DB, data)
	if err != nil {
		return err
	}

	// Remove old held promotions after new one is added
	for _, promoID := range promotionIDs {
		err := models.RemoveHeldProductPromotions(models.DB, promoID)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetExpensiveAndCheapProducts retrieves the most expensive and cheapest products.
// It utilizes caching for performance.
//
// @Summary      Get expensive and cheap products
// @Description  Retrieve the most expensive and cheapest products
// @Tags         Products
// @Produce      json
// @Success      200         {object}  dtos.ExpensiveCheapProduct
// @Failure      500         {object}  dtos.ErrorResponse
// @Router       /api/products/expensive-cheap [get]
func GetExpensiveAndCheapProducts(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Define cache key for expensive/cheap products
	cacheKey := "expensiveandcheapproducts"
	var products *dtos.ExpensiveCheapProduct
	var cachedProducts *dtos.ExpensiveCheapProduct

	// Try to retrieve from Redis cache
	_ = utils.GetCache(cacheKey, &cachedProducts)
	if cachedProducts == nil {
		// Cache miss - fetch from database
		var err error
		products, err = models.GetExpensiveAndCheapProducts(models.DB)
		if err != nil {
			// Database query failed, return error response
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to fetch expensive and cheapest products",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
			})
			return
		}
		// Cache the fetched products (no expiration set - uses default)
		_ = utils.SetCache(cacheKey, products)
	} else {
		// Cache hit - use cached data
		products = cachedProducts
	}

	// Return successful response with expensive and cheap products
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Expensive and cheapest products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   products,
		Message:   "Expensive and cheapest products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
