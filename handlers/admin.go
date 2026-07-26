package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/csv"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var uploadImageError = "Failed to upload image"

// This provides the code for products admin, products categories, products reviews and product bundles functionalities that require admin authorization

// CreateCategoryHandler creates a new product category.
// It supports optional image upload.
// This endpoint is restricted to administrators.
//
// @Summary      Add a new category
// @Description  Create a new product category with an optional image
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        name         formData  string  true  "Category Name"
// @Param        description  formData  string  true  "Category Description"
// @Param        parent_id    formData  string  false "Parent Category ID"
// @Param        image        formData  file    false "Category Image"
// @Success      201          {object}  dtos.Category
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/products/categories [post]
func CreateCategoryHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Categories", "categories.create"); !ok {
		return
	}

	// Upload image if present
	url, err := utils.ParseAndUploadFile(c.Request, "image", 20)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to upload image : " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
		})
		return
	}

	// Build DTO
	req := dtos.CreateCategory{
		Image:       url,
		Name:        c.Request.FormValue("name"),
		Description: c.Request.FormValue("description"),
		ParentID:    utils.StringPtr(c.Request.FormValue("parent_id")),
	}
	// Validate request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Categories") {
		return
	}

	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	category, err := models.AddNewCategory(models.DB, req, tenantID)
	if err != nil {
		log.Printf("Error adding new category: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to add new category",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Invalidate categories cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	// Respond success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   category,
		Message:   "Category created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary,
	})
}

// UpdateCategoryHandler updates an existing category.
// It supports updating name, description, parent ID, and image.
// This endpoint is restricted to administrators.
//
// @Summary      Update a category
// @Description  Update an existing product category
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        category_id  path      string  true  "Category ID"
// @Param        name         formData  string  false "Category Name"
// @Param        description  formData  string  false "Category Description"
// @Param        parent_id    formData  string  false "Parent Category ID"
// @Param        image        formData  file    false "Category Image"
// @Success      200          {object}  dtos.Category
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/categories/{category_id} [patch]
func UpdateCategoryHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Categories", "categories.update"); !ok {
		return
	}

	// Parse multipart form (20 MB max)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to parse form: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
		})
		return
	}

	// Handle optional image upload
	var imageURL string
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Categories",
					Description: err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   uploadImageError,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request: c.Request,
			})
			return
		}
		imageURL = url
	}

	// Build DTO (image is optional)
	req := dtos.UpdateCategoryPayload{
		Image:       &imageURL,
		Name:        c.Request.FormValue("name"),
		Description: c.Request.FormValue("description"),
		ParentID:    utils.StringPtr(c.Request.FormValue("parent_id")),
	}

	// Validate request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Categories") {
		return
	}

	// Get category ID from URL
	id := c.Param("category_id")
	// Update category
	category, err := models.UpdateCategory(models.DB, id, req)
	if err != nil {
		log.Printf("Error updating category: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to update category with id " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Invalidate cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")

	// Respond success
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category with ID " + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   category,
		Message:   "Category updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary,
	})
}

// DeleteCategoryHandler deletes an existing category.
// This endpoint is restricted to administrators.
//
// @Summary      Delete a category
// @Description  Delete a category by its ID
// @Tags         Admin
// @Produce      json
// @Param        category_id  path      string  true  "Category ID"
// @Success      200          {object}  map[string]interface{}
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      409          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/products/category/{category_id} [delete]
func DeleteCategoryHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Categories", "categories.delete")
	if !ok {
		return
	}
	id := c.Param("category_id")
	err := models.DeleteCategory(models.DB, id)
	if err != nil {
		log.Printf("Error deleting product %s", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Categories",
				Description: "Failed to delete category with id " + id,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cache
	_ = utils.DeleteCache("category_data")
	_ = utils.DeleteCacheByPrefix("categories_products")
	_ = utils.DeleteCacheByPrefix("categories_products_pagination")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Categories",
			Description: "Category with ID " + id + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Category deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary})
}

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
			Request: c.Request,
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
			Request: c.Request,
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
		Request: c.Request,
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
			Request: c.Request,
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
		Request: c.Request,
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
// @Success      200         {object}  map[string]interface{}
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
			Request: c.Request,
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
		Request: c.Request,
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
// @Success      200     {object}  map[string]interface{}
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
			Request: c.Request,
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
		Request: c.Request,
		RawBody:   requestSummary})
}

// UploadImageHandler uploads an image to GCS and returns the URL.
// This endpoint is restricted to administrators.
//
// @Summary      Upload an image
// @Description  Upload an image file to Google Cloud Storage
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image File"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/upload [post]
func UploadImageHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(c.Request, "image", 10)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to upload image: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
		})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Image with URL " + url + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   url,
		Message:   "Image uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary,
	})
}

// UploadImageHandler2 uploads an image to GCS and returns the URL.
// This is a duplicate of UploadImageHandler, likely for testing or legacy reasons.
// This endpoint is restricted to administrators.
//
// @Summary      Upload an image (Alternative)
// @Description  Upload an image file to Google Cloud Storage
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image File"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/upload2 [post]
func UploadImageHandler2(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.create")
	if !ok {
		return
	}
	url, err := utils.ParseAndUploadFile(c.Request, "image", 10)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to upload image: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
		})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Image with URL " + url + " uploaded successfully",
			Code:        http.StatusOK,
		},
		Payload:   url,
		Message:   "Image uploaded successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary,
	})
}

// GetAllSubscribersHandler retrieves a paginated list of newsletter subscribers.
//
// @Summary      Get all subscribers
// @Description  Retrieve a paginated list of newsletter subscribers with optional filtering
// @Tags         Admin
// @Produce      json
// @Param        page        query     int     false  "Page number"
// @Param        size        query     int     false  "Page size"
// @Param        q           query     string  false  "Search query"
// @Param        start_date  query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string  false  "End date (YYYY-MM-DD)"
// @Success      200         {object}  map[string]interface{}
// @Failure      401         {object}  dtos.ErrorResponse
// @Failure      500         {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/subscribers [get]
func GetAllSubscribersHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Subscribers", "subscribers.view"); !ok {
		return
	}

	// Parse pagination and filters
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	q := c.Query("q")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	offset := (page - 1) * size

	// Fetch subscribers
	subscribers, meta, err := models.GetAllSubscribers(models.DB, size, offset, q, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching subscribers: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Subscribers",
				Description: "Failed to fetch subscribers",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Respond with data
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Subscribers",
			Description: "Subscribers fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"subscribers": subscribers,
			"pagination":  meta,
		},
		Message:   "Subscribers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request: c.Request,
		RawBody:   requestSummary,
	})
}

// DownloadSubscribersCSVHandler downloads the subscriber list as a CSV.
//
// @Summary      Download subscribers CSV
// @Description  Download a CSV file containing all subscribers or filtered results
// @Tags         Admin
// @Produce      text/csv
// @Param        q           query     string  false  "Search query"
// @Param        start_date  query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end_date    query     string  false  "End date (YYYY-MM-DD)"
// @Success      200         {file}    file
// @Security     BearerAuth
// @Router       /api/admin/subscribers/csv [get]
func DownloadSubscribersCSVHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user is admin
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Subscribers", "subscribers.view"); !ok {
		return
	}

	// Parse filters (ignore pagination for CSV export)
	q := c.Query("q")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Fetch subscribers - using a large limit for export
	subscribers, _, err := models.GetAllSubscribers(models.DB, 1000000, 0, q, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching subscribers for CSV: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Subscribers",
				Description: "Failed to fetch subscribers for CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request: c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Set headers for CSV download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=subscribers.csv")

	// Initialize CSV writer
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header row
	header := []string{"Email", "Date Subscribed"}
	if err := writer.Write(header); err != nil {
		log.Printf("Error writing CSV header: %v", err)
		return
	}

	// Write data rows
	for _, s := range subscribers {
		row := []string{
			s.Email,
			s.CreatedAt,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Error writing CSV row: %v", err)
			return
		}
	}
}
