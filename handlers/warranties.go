// Package handlers provides HTTP request handlers for warranty management.
// This file contains handlers for managing warranty types and product warranty associations,
// including CRUD operations for warranty categories (e.g., manufacturer warranty, extended warranty)
// and linking warranty options to specific products. Essential for product protection plans and customer service.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// CreateWarrantType creates a new warranty type in the system.
// Admin-only operation for adding warranty categories like manufacturer warranty,
// extended warranty, or service plans with duration and coverage details.
//
// @Summary      Create warranty type
// @Description  Create a new warranty type/category with duration and terms (admin only)
// @Tags         Warranties
// @Accept       json
// @Produce      json
// @Param        warranty  body      dtos.CreateWarrantyTypeRequest  true  "Warranty type details"
// @Success      201       {object}  dtos.SuccessResponse            "Warranty type created successfully"
// @Failure      400       {object}  dtos.ErrorResponse              "Invalid request or creation failed"
// @Failure      401       {object}  dtos.ErrorResponse              "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/warranty-types [post]
func CreateWarrantType(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can create warranty types)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Decode and parse JSON request body with warranty type details
	req, ok := DecodeRequestBody[dtos.CreateWarrantyTypeRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (name, duration, coverage terms)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Create warranty type record in database
	err := models.CreateWarrantType(*req)
	if err != nil {
		// Warranty type creation failed (duplicate name, invalid data, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create warranty",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response with 201 Created status
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Warranty created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Warranty created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetAllWarrantyTypes retrieves all available warranty types.
// Public endpoint for displaying warranty options available for products.
// Returns all warranty categories with duration, coverage, and pricing information.
//
// @Summary      List all warranty types
// @Description  Retrieve all warranty types/categories available in the system
// @Tags         Warranties
// @Produce      json
// @Success      200  {object}  dtos.SuccessResponse  "Warranty types retrieved successfully"
// @Failure      500  {object}  dtos.ErrorResponse    "Failed to retrieve warranty types"
// @Router       /api/warranty-types [get]
func GetAllWarrantyTypes(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Fetch all warranty types from database
	warrantyTypes, err := models.GetAllWarrantTypes()
	if err != nil {
		// Database query failed
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get warranty types",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return warranty types list with success status
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Warranty types retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   warrantyTypes,
		Message:   "Warranty types retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateWarrantType modifies an existing warranty type's information.
// Admin-only operation for updating warranty duration, coverage terms, or pricing.
// Essential for maintaining accurate warranty offerings.
//
// @Summary      Update warranty type
// @Description  Update warranty type information like duration, coverage, or pricing (admin only)
// @Tags         Warranties
// @Accept       json
// @Produce      json
// @Param        warranty_type_id  path      string                  true  "Warranty Type ID"
// @Param        warranty          body      dtos.WarrantyType       true  "Updated warranty type details"
// @Success      200               {object}  dtos.SuccessResponse    "Warranty type updated successfully"
// @Failure      400               {object}  dtos.ErrorResponse      "Invalid request or update failed"
// @Failure      401               {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      404               {object}  dtos.ErrorResponse      "Warranty type not found"
// @Security     BearerAuth
// @Router       /api/admin/warranty-types/{warranty_type_id} [patch]
func UpdateWarrantType(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can update warranty types)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.update")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with updated warranty details
	req, ok := DecodeRequestBody[dtos.WarrantyType](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract warranty type ID from URL path parameters
	warrantID := mux.Vars(r)["warranty_type_id"]
	// Update warranty type information in database
	err := models.UpdateWarrantType(warrantID, *req)
	if err != nil {
		// Update failed (warranty type not found, invalid data, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to update warranty type with ID " + warrantID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Warranty type with ID " + warrantID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warranty type updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteWarrantType removes a warranty type from the system.
// Admin-only operation for deactivating or removing warranty offerings.
//
// @Summary      Delete warranty type
// @Description  Delete or deactivate a warranty type (admin only)
// @Tags         Warranties
// @Produce      json
// @Param        warranty_type_id  path      string                  true  "Warranty Type ID"
// @Success      200               {object}  dtos.SuccessResponse    "Warranty type deleted successfully"
// @Failure      401               {object}  dtos.ErrorResponse      "Admin authorization required"
// @Failure      404               {object}  dtos.ErrorResponse      "Warranty type not found"
// @Security     BearerAuth
// @Router       /api/admin/warranty-types/{warranty_type_id} [delete]
func DeleteWarrantType(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete warranty types)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract warranty type ID from URL path parameters
	warrantID := mux.Vars(r)["warranty_type_id"]
	// Delete warranty type from database (may be soft delete)
	err := models.DeleteWarrantType(warrantID)
	if err != nil {
		// Deletion failed (warranty type not found, has dependencies, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete warranty type with ID " + warrantID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Warranty type with ID " + warrantID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Warranty type deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddProductWarranties associates warranty options with a specific product.
// Admin-only operation for linking warranty types to products, enabling customers
// to purchase warranty protection during checkout. Essential for product warranty offerings.
//
// @Summary      Add warranties to product
// @Description  Associate warranty types with a specific product (admin only)
// @Tags         Warranties
// @Accept       json
// @Produce      json
// @Param        warranties  body      dtos.AddProductWarrantiesRequest  true  "Product ID and warranty type IDs"
// @Success      200         {object}  dtos.SuccessResponse              "Product warranties added successfully"
// @Failure      400         {object}  dtos.ErrorResponse                "Invalid request or association failed"
// @Failure      401         {object}  dtos.ErrorResponse                "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/products/warranties [post]
func AddProductWarranties(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can add product warranties)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.create")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body with product ID and warranty type IDs
	req, ok := DecodeRequestBody[dtos.AddProductWarrantiesRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (product ID and warranty type IDs)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Create product-warranty associations in database
	err := models.AddProductWarranties(*req)
	if err != nil {
		// Association failed (product not found, invalid warranty types, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to add product warranties",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return success response
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product warranties added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Product warranties added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
