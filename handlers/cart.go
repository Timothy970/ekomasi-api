package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"

	"github.com/gorilla/mux"
)

var notAuthenticated = "User not authenticated"

const cartCacheDuration = 5 * time.Minute

// Create a cart handler
// @Summary Create Cart
// @Description Create a user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Success 200 {object} dtos.AddToCartResponse
// @Failure 400 {object} dtos.ErrorResponse
// @Failure 500 {object} dtos.ErrorResponse
// @Security BearerAuth
// @Router /api/cart [post]
func CreateCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.CreateCartRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	cartID, err := models.CreateCart(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   map[string]interface{}{"cart_id": cartID},
		Message:   "Cart created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get a cart for user
// @Summary Get Cart
// @Description Get a user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Success 200 {object} dtos.AddToCartResponse
// @Failure 400 {object} dtos.ErrorResponse
// @Failure 500 {object} dtos.ErrorResponse
// @Security BearerAuth
// @Router /api/cart [get]
func GetUserCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   notAuthenticated,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	cartID, err := models.GetUserCart(user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   map[string]interface{}{"cart_id": cartID},
		Message:   "Cart fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddToCartHandler handles adding a product to the cart
// @Summary Add to Cart
// @Description Add a product to user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param cart body dtos.AddToCartRequest true "Cart item"
// @Success 200 {object} dtos.AddToCartResponse
// @Failure 400 {object} dtos.ErrorResponse
// @Failure 500 {object} dtos.ErrorResponse
// @Security BearerAuth
// @Router /api/cart/add [post]
func AddToCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.AddToCartRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	if err := models.InsertCartItem(req.CartID, req.ProductID, req.Quantity); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      404,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Product added to cart successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ViewCartHandler returns the contents of a user's cart
// @Summary View Cart
// @Description Retrieve current items in cart
// @Tags Cart
// @Produce json
// @Success 200 {object} dtos.ViewCartResponse
// @Failure 400 {object} dtos.ErrorResponse
// @Failure 500 {object} dtos.ErrorResponse
// @Security BearerAuth
// @Router /api/cart/view/{cart_id} [get]
func ViewCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	cartID := mux.Vars(r)["cart_id"]
	items, err := models.GetCartItems(cartID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	discount := 0.0
	total := 0.0
	for _, item := range items {
		//check if any product has a discount
		productDiscount, err := models.GetProductPromotionData(item.ProductID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		if productDiscount.Type != "" {
			discountAmount, err := calculateDifferentDiscountTypes(&productDiscount, item)
			if err != nil {
				log.Printf("Error calculating discount: %v", err)
			} else {
				fmt.Printf("Discount for %s: %.2f\n", item.ProductID, discountAmount)
				discount += discountAmount
			}
		}
		total += float64(item.Quantity) * item.Price
	}
	res := dtos.ViewCartResponse{
		CartItems: items,
		Total:     total,
		Final:     total,
		Discount:  discount,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   res,
		Message:   "Cart Items fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func calculateDifferentDiscountTypes(promo *dtos.PromotionData, item dtos.CartItem) (float64, error) {
	switch promo.Type {
	case "Percentage":
		// Apply a percentage discount to total item price
		return (promo.Value / 100) * float64(item.Quantity) * item.Price, nil

	case "Fixed":
		// Apply a fixed amount discount (flat rate)
		// Split proportionally if needed; here we apply it fully if item subtotal > discount
		subtotal := float64(item.Quantity) * item.Price
		if subtotal > promo.Value {
			return promo.Value, nil
		}
		return subtotal, nil

	case "BOGO":
		// Buy One Get One Free
		// For every two items, one is free
		if item.Quantity > 1 {
			freeItems := item.Quantity / 2
			return float64(freeItems) * item.Price, nil
		} else {
			return 0, nil
		}

	case "FreeShipping":
		// Delivery fee is waived - will be handled at order creation
		// Return 0 as this doesn't affect product price directly
		return 0, nil

	case "Tiered":
		// Example tiered logic based on quantity
		if item.Quantity >= 10 {
			return 0.20 * float64(item.Quantity) * item.Price, nil // 20% off
		} else if item.Quantity >= 5 {
			return 0.10 * float64(item.Quantity) * item.Price, nil // 10% off
		}
		return 0, nil

	default:
		return 0, fmt.Errorf("unsupported promotion type: %s", promo.Type)
	}
}

// func checkIfProductHasDiscount(item dtos.CartItem) (map[string]interface{}, error) {
// 	discountData
// }

// UpdateCartItemHandler updates quantity of an item in the cart
// @Summary Update Cart Item
// @Description Update a cart item quantity
// @Tags Cart
// @Accept json
// @Produce json
// @Param item body dtos.UpdateCartItemRequest true "Update cart item"
// @Success 200 {object} dtos.UpdateCartItemResponse
// @Failure 400 {object} dtos.ErrorResponse
// @Failure 500 {object} dtos.ErrorResponse
// @Security BearerAuth
// @Router /api/cart/update/{cart_id} [put]
func UpdateCartItemHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	cartID := mux.Vars(r)["cart_id"]

	req, ok := DecodeRequestBody[dtos.UpdateCartItemRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	if err := models.UpdateCartItem(cartID, req.ProductID, req.Quantity); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Cart item updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// RemoveFromCartHandler removes an item from the cart
// @Summary Remove Cart Item
// @Description Remove a product from user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param item body dtos.RemoveFromCartRequest true "Remove cart item"
// @Success 200 {object} dtos.RemoveFromCartResponse
// @Failure 400 {object} dtos.ErrorResponse
// @Failure 500 {object} dtos.ErrorResponse
// @Security BearerAuth
// @Router /api/cart/remove/{cart_id} [delete]
func RemoveFromCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	req, ok := DecodeRequestBody[dtos.RemoveFromCartRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	cartID := mux.Vars(r)["cart_id"]

	if err := models.DeleteCartItem(cartID, req.ProductID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Product removed from cart successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ApplyCouponHandler applies a discount coupon to the cart
// @Summary Apply Coupon
// @Description Apply a discount coupon to user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param coupon body dtos.CouponRequest true "Coupon data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} dtos.ErrorResponse
// @Security BearerAuth
// @Router /api/cart/apply-coupon [post]
// to do: refine this
func ApplyCouponHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.CouponRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusUnauthorized,
			Message:   notAuthenticated,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//validate coupon
	couponData, err := models.ValidateCoupon(req.CouponCode)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	//get user cart
	items, err := models.GetCartItems(user.ID)
	if err != nil {
		log.Printf("Error fetching cart items: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to fetch cart items",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	total := 0.0
	for _, item := range items {
		total += float64(item.Quantity) * item.Price
	}
	discount := couponData / 100 * total
	final := total - discount
	res := dtos.ViewCartResponse{
		CartItems: items,
		Total:     total,
		Discount:  discount,
		Final:     final,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   res,
		Message:   "Coupon applied successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
