package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateProductHandler creates a new product.
// This endpoint is restricted to administrators.
//
// @Summary      Add a new product
// @Description  Create a new product with initial stock quantity set to 0
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        product  body      dtos.CreateProduct  true  "Product Details"
// @Success      201      {object}  dtos.Product
// @Failure      400      {object}  dtos.ErrorResponse
// @Failure      401      {object}  dtos.ErrorResponse
// @Failure      409      {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products [post]
func CreateProductHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "User not validated or authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateProduct](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}
	//when adding a new product, stock quantity is always 0
	req.StockQuantity = 0
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	product, err := models.AddNewProduct(models.DB, *req, authuser.ID, tenantID)
	if err != nil {
		log.Printf("Error for adding new product %s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add new product",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cache
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	_ = utils.DeleteCache("expensiveandcheapproducts")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   product,
		Message:   "Product created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateProductHandler updates an existing product.
// This endpoint is restricted to administrators.
//
// @Summary      Update product
// @Description  Update an existing product by ID
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        product_id  path      string              true  "Product ID"
// @Param        product     body      dtos.CreateProduct  true  "Product Details"
// @Success      200         {object}  dtos.Product
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      409         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/{product_id} [patch]
func UpdateProductHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.update")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateProduct](c, requestSummary, start)
	if !ok {
		return
	}
	productID := c.Param("product_id")
	// update the product
	updatedProduct, err := models.UpdateProductByID(models.DB, productID, *req)

	if err != nil {
		log.Printf("Error for updating new product %s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product with id " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate cache
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product with ID " + productID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   updatedProduct,
		Message:   "Product updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteProductHandler deletes a product.
// This endpoint is restricted to administrators.
//
// @Summary      Delete product
// @Description  Delete a product by ID
// @Tags         Admin
// @Produce      json
// @Param        product_id  path      string  true  "Product ID"
// @Success      200         {object}  map[string]any
// @Failure      400         {object}  dtos.ErrorResponse
// @Failure      409         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/{product_id} [delete]
func DeleteProductHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.delete")
	if !ok {
		return
	}
	productID := c.Param("product_id")
	//delete product
	err := models.DeleteProductByID(models.DB, productID)
	if err != nil {
		log.Printf("Error for deleting product %s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete product with id " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate cache
	_ = utils.DeleteCacheByPrefix("products_page_")
	_ = utils.DeleteCacheByPrefix("pagination_page_")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product with ID " + productID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AddCoupon creates a new coupon.
// This endpoint is restricted to administrators.
//
// @Summary      Add a new coupon
// @Description  Create a new promotional coupon
// @Tags         Promotions
// @Accept       json
// @Produce      json
// @Param        coupon  body      dtos.PromoCode  true  "Coupon Details"
// @Success      200     {object}  map[string]any
// @Failure      400     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/coupons [post]
func AddCoupon(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	req, ok := DecodeRequestBody[dtos.PromoCode](c, requestSummary, start)
	if !ok {
		return
	}
	// user, ok := middleware.UserFromContext(c.Request.Context())
	// if !ok {
	// 	utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
	// 		Code:      http.StatusUnauthorized,
	// 		Message:   notAuthenticated,
	// 		TimeTaken: time.Since(start),
	// 		Function:  utils.GetCurrentFuncName(),
	// 		Request: c.Request,
	// 		RawBody:   requestSummary})
	// 	return
	// }
	if err := models.CreateCoupon(models.DB, *req); err != nil {
		log.Printf("Error adding item to cart: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to create coupon",
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to create coupon",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Coupon created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Coupon created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
