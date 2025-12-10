package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

//CRUD operations for bundle products

// Get all bundle products
func GetBundleProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	bundles, pagination, err := models.GetBundleProducts(limit, page)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch bundles",
				Code:        http.StatusInternalServerError,
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
			Module:      "Products",
			Description: "Bundles fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"bundles": bundles, "pagination": pagination},
		Message:   "Bundles fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Get just one bundle product by ID
func GetBundleByIDProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	bundleID := mux.Vars(r)["bundle_id"]
	bundles, err := models.GetBundleByIDProducts(bundleID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to fetch bundle",
				Code:        http.StatusInternalServerError,
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
			Module:      "Products",
			Description: "Bundle fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   bundles,
		Message:   "Bundle fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Create a new bundle
func CreateBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	authuser, _ := middleware.UserFromContext(r.Context())

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse form data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	// Get image file
	file, header, err := r.FormFile("image")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Image is required when creating a bundle",
				Code:        http.StatusBadRequest,
			},
			Message:   "Image is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	defer file.Close()
	// Upload image to GCS
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	req := &dtos.Bundle{
		Name:        r.FormValue("bundle_name"),
		Description: r.FormValue("bundle_description"),
		Price:       func() float64 { p, _ := strconv.ParseFloat(r.FormValue("bundle_price"), 64); return p }(),
		Image:       url,
		Products: func() []dtos.BundleProducts {
			productsStr := r.FormValue("products")
			if productsStr == "" {
				return []dtos.BundleProducts{}
			}

			var products []dtos.BundleProducts
			if err := json.Unmarshal([]byte(productsStr), &products); err != nil {
				// you may want to handle error properly instead of swallowing it
				return []dtos.BundleProducts{}
			}
			return products
		}(),
		KeepSelling: func() *bool {
			ks := strings.ToLower(r.FormValue("keep_selling"))

			// default = true (empty or key missing)
			defaultValue := true

			if ks == "" {
				return &defaultValue
			}

			switch ks {
			case "true":
				v := true
				return &v
			case "false":
				v := false
				return &v
			}

			// any unexpected value → default to true
			return &defaultValue
		}(),

		CompareAtPrice: func() *float64 {
			cp, _ := strconv.ParseFloat(r.FormValue("compare_at_price"), 64)
			if cp == 0 {
				return nil
			}
			return &cp
		}(),
		StockQuantity: func() int {
			sq, _ := strconv.Atoi(r.FormValue("stock_quantity"))
			return sq
		}(),
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}
	err = models.CreateBundle(*req, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create product bundle",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("products_bundles_page_")
	utils.DeleteCacheByPrefix("bundles_pagination_page_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product bundle created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Created product bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// Update an existing bundle
func UpdateBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	bundleID := mux.Vars(r)["bundle_id"]
	// Parse multipart form (20 MB max)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to parse form data: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to parse form: " + err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Handle optional image upload
	var imageURL string
	if file, header, err := r.FormFile("image"); err == nil {
		defer file.Close()
		url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Failed to upload image to storage :" + err.Error(),
					Code:        http.StatusInternalServerError,
				},
				Message:   "Failed to upload image",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
			})
			return
		}
		imageURL = url
	} else {
		// If no file uploaded, check for image URL string in form value
		imageURL = r.FormValue("image")
	}

	req := &dtos.Bundle{
		Name:        r.FormValue("bundle_name"),
		Description: r.FormValue("bundle_description"),
		Price:       func() float64 { p, _ := strconv.ParseFloat(r.FormValue("bundle_price"), 64); return p }(),
		Image:       imageURL,
		Products: func() []dtos.BundleProducts {
			productsStr := r.FormValue("products")
			if productsStr == "" {
				return []dtos.BundleProducts{}
			}

			var products []dtos.BundleProducts
			if err := json.Unmarshal([]byte(productsStr), &products); err != nil {
				// you may want to handle error properly instead of swallowing it
				return []dtos.BundleProducts{}
			}
			return products
		}(),
		KeepSelling: func() *bool {
			ks := strings.ToLower(r.FormValue("keep_selling"))

			// default = true (empty or key missing)
			defaultValue := true

			if ks == "" {
				return &defaultValue
			}

			switch ks {
			case "true":
				v := true
				return &v
			case "false":
				v := false
				return &v
			}

			// any unexpected value → default to true
			return &defaultValue
		}(),

		CompareAtPrice: func() *float64 {
			cp, _ := strconv.ParseFloat(r.FormValue("compare_at_price"), 64)
			if cp == 0 {
				return nil
			}
			return &cp
		}(),
		StockQuantity: func() int {
			sq, _ := strconv.Atoi(r.FormValue("stock_quantity"))
			return sq
		}(),
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}
	err := models.UpdateBundle(*req, bundleID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update product bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
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
			Module:      "Products",
			Description: "Product bundle with ID " + bundleID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product bundle updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete a bundle
func DeleteBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	bundleID := mux.Vars(r)["bundle_id"]
	err := models.DeleteBundle(bundleID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete product bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
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
			Module:      "Products",
			Description: "Product bundle with ID " + bundleID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product bundle deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// add products to a bundle
func AddProductsToBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[[]dtos.BundleProducts](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}
	err := models.AddProductsToBundle(*req, mux.Vars(r)["bundle_id"])
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product(s) to bundle with ID " + mux.Vars(r)["bundle_id"],
				Code:        http.StatusInternalServerError,
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
			Module:      "Products",
			Description: "Product(s) added to bundle with ID " + mux.Vars(r)["bundle_id"] + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product(s) added to bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Remove products from a bundle
func RemoveProductsFromBundleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	bundleID := mux.Vars(r)["bundle_id"]
	req, ok := DecodeRequestBody[dtos.AddProductsToBundle](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}
	err := models.RemoveProductsFromBundle(*req, bundleID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to remove product(s) from bundle with ID " + bundleID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to remove product(s) from bundle",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("products_bundles_page_")
	utils.DeleteCacheByPrefix("bundles_pagination_page_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product(s) removed from bundle with ID " + bundleID + " successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product(s) removed from bundle successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
