package handlers

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"

	"github.com/gorilla/mux"
)

var (
	notAuthenticated     = "User not authenticated"
	failedToGetCartItems = "Failed to get cart items for Cart ID "
)

// CreateCartHandler creates a new shopping cart for the user.
//
// @Summary      Create Cart
// @Description  Create a new shopping cart for the user.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        cart  body      dtos.CreateCartRequest  true  "Create Cart Request"
// @Success      200   {object}  dtos.AddToCartResponse
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart [post]
func CreateCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.CreateCartRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	// if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
	// 	return
	// }

	// Create cart in database
	cartID, err := models.CreateCart(models.DB, *req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to create cart",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with created cart ID
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Cart created successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"cart_id": cartID},
		Message:   "Cart created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetUserCartHandler retrieves the cart for the authenticated user.
//
// @Summary      Get Cart
// @Description  Retrieve the shopping cart for the currently authenticated user.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Success      200  {object}  dtos.AddToCartResponse
// @Failure      400  {object}  dtos.ErrorResponse
// @Failure      403  {object}  dtos.ErrorResponse
// @Failure      500  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart [get]
func GetUserCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Get authenticated user from context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "User not authenticated to fetch cart",
				Code:        http.StatusForbidden,
			},
			Message:   notAuthenticated,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch user's cart ID
	cartID, err := models.GetUserCart(models.DB, user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to fetch user cart with user ID" + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with cart ID
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Cart with ID " + cartID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"cart_id": cartID},
		Message:   "Cart fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// AddToCartHandler adds a product to the user's cart.
//
// @Summary      Add to Cart
// @Description  Add a product item to the user's shopping cart.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        cart  body      dtos.AddToCartRequest  true  "Cart Item Details"
// @Success      200   {object}  dtos.AddToCartResponse
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      404   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart/add [post]
func AddToCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.AddToCartRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Cart") {
		return
	}

	// Insert item into cart
	if err := models.InsertCartItem(models.DB, req.CartID, req.ProductID, req.Quantity, req.VariationSKU); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to add item to cart with Cart ID " + req.CartID,
				Code:        404,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch updated cart items
	res, err := getCartItemsByCartID(req.CartID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: failedToGetCartItems + req.CartID,
				Code:        404,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with updated cart items
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Product with ID " + req.ProductID + " added to cart successfully with Cart ID " + req.CartID,
			Code:        http.StatusOK,
		},
		Payload:   res,
		Message:   "Product added to cart successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ViewCartHandler retrieves the contents of a specific cart.
//
// @Summary      View Cart
// @Description  Retrieve current items in the specified cart.
// @Tags         Cart
// @Produce      json
// @Param        cart_id      path      string  true   "Cart ID"
// @Param        location_id  query     int     false  "Location ID for delivery charge calculation"
// @Success      200          {object}  dtos.ViewCartResponse
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart/view/{cart_id} [get]
func ViewCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract cart ID from path variables
	cartID := mux.Vars(r)["cart_id"]
	locationID := r.URL.Query().Get("location_id")

	// Fetch cart items
	res, err := getCartItemsByCartID(cartID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: failedToGetCartItems + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Calculate delivery charge if location ID is provided
	if locationID != "" {
		locationIDInt, _ := strconv.Atoi(locationID)
		if locationIDInt != 0 {
			loc, err := models.GetLocationByID(models.DB, locationIDInt)
			if err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Cart",
						Description: "Failed to fetch location with ID " + locationID,
						Code:        http.StatusInternalServerError,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary})
				return
			}
			res.DeliverCharge = loc.Charge
			res.TotalAmount += loc.Charge

		}
	}

	// Respond with cart items
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Cart Items fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   res,
		Message:   "Cart Items fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func getCartItemsByCartID(cartID string) (dtos.ViewCartResponse, error) {
	items, err := models.GetCartItems(models.DB, cartID)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}
	estimatedTax, _ := models.GetEstimatedTax(models.DB)
	if estimatedTax <= 0 {
		estimatedTax = 16.0
	}
	discount := 0.0
	subtotal := 0.0
	total := 0.0
	for _, item := range items {
		discountValue, discountType, err := models.GetProductDiscount(models.DB, item.Product.ID)
		if err != nil {
			log.Printf("Error fetching product discount: %v", err)
			return dtos.ViewCartResponse{}, errors.New("failed to fetch product discount")
		}
		if discountType != "" && discountValue > 0 {
			item.Product.Price = calculateNewPriceWithDiscount(item.Product.Price, discountValue, discountType)
		}
		//product price including VAT
		priceIncVAT := float64(item.Quantity) * item.Product.Price
		//product price excluding VAT
		priceExclVAT := priceIncVAT / (1 + estimatedTax/100)
		//Calculate subtotal (sum of prices before VAT)
		subtotal += priceExclVAT
		//total should be price including VAT
		total += priceIncVAT
		//check if any product has a discount
		productDiscount, err := models.GetProductPromotionData(models.DB, item.Product.ID)
		if err != nil {
			return dtos.ViewCartResponse{}, err
		}
		if productDiscount.Type != "" {
			discountAmount, err := calculateDifferentDiscountTypes(&productDiscount, item)
			if err != nil {
				log.Printf("Error calculating discount: %v", err)
			} else {
				fmt.Printf("Discount for %s: %.2f\n", item.Product.ID, discountAmount)
				discount += discountAmount
			}
		}
	}

	//get estimated tax
	estimatedTaxValue := total - subtotal
	// Round up values to the next whole number
	subtotal = math.Ceil(subtotal)
	total = math.Ceil(total)
	discount = math.Ceil(discount)
	estimatedTaxValue = math.Ceil(estimatedTaxValue)
	res := dtos.ViewCartResponse{
		CartItems:    items,
		SubTotal:     subtotal,         //this is the subtotal
		TotalAmount:  total - discount, //this is the total, should be less the discount
		Discount:     discount,
		EstimatedTax: estimatedTaxValue,
	}
	return res, nil
}

// helper function to calculate discount based on different promotion types
func calculateNewPriceWithDiscount(price, discountValue float64, discountType string) float64 {
	switch strings.ToLower(discountType) {
	case "percentage":
		return price * (1 - discountValue/100)
	case "fixed":
		return price - discountValue
	default:
		return price
	}
}

func calculateDifferentDiscountTypes(promo *dtos.PromotionData, item dtos.CartItem) (float64, error) {
	switch strings.ToLower(promo.Type) {
	case "percentage":
		// Apply a percentage discount to total item price
		return (promo.Value / 100) * float64(item.Quantity) * item.Product.Price, nil

	case "fixed":
		// Apply a fixed amount discount (flat rate)
		// Split proportionally if needed; here we apply it fully if item subtotal > discount
		subtotal := float64(item.Quantity) * item.Product.Price
		if subtotal > promo.Value {
			return promo.Value, nil
		}
		return subtotal, nil

	case "bogo":
		// Buy One Get One Free
		// For every two items, one is free
		if item.Quantity > 1 {
			freeItems := item.Quantity / 2
			return float64(freeItems) * item.Product.Price, nil
		} else {
			return 0, nil
		}

	case "freeshipping":
		// Delivery fee is waived - will be handled at order creation
		// Return 0 as this doesn't affect product price directly
		return 0, nil

	case "tiered":
		// Example tiered logic based on quantity
		if item.Quantity >= 10 {
			return 0.20 * float64(item.Quantity) * item.Product.Price, nil // 20% off
		} else if item.Quantity >= 5 {
			return 0.10 * float64(item.Quantity) * item.Product.Price, nil // 10% off
		}
		return 0, nil

	default:
		return 0, fmt.Errorf("unsupported promotion type: %s", promo.Type)
	}
}

// func checkIfProductHasDiscount(item dtos.CartItem) (map[string]interface{}, error) {
// 	discountData
// }

// UpdateCartItemHandler updates the quantity of a specific item in the cart.
//
// @Summary      Update Cart Item
// @Description  Update the quantity of an item in the shopping cart.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        cart_id  path      string                     true  "Cart ID"
// @Param        item     body      dtos.UpdateCartItemRequest true  "Update Cart Item Details"
// @Success      200      {object}  dtos.UpdateCartItemResponse
// @Failure      400      {object}  dtos.ErrorResponse
// @Failure      500      {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart/update/{cart_id} [patch]
func UpdateCartItemHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract cart ID from path variables
	cartID := mux.Vars(r)["cart_id"]

	// Decode request body
	req, ok := DecodeRequestBody[dtos.UpdateCartItemRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Update cart item in database
	if err := models.UpdateCartItem(models.DB, cartID, req.ProductID, req.Quantity); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to update cart item with Cart ID " + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch updated cart items
	res, err := getCartItemsByCartID(cartID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: failedToGetCartItems + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with updated cart items
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Cart item updated successfully with Cart ID " + cartID,
			Code:        http.StatusOK,
		},
		Payload:   res,
		Message:   "Cart item updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// RemoveFromCartHandler removes a specific item from the cart.
//
// @Summary      Remove Cart Item
// @Description  Remove a product from the user's shopping cart.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        cart_id  path      string                     true  "Cart ID"
// @Param        item     body      dtos.RemoveFromCartRequest true  "Remove Cart Item Details"
// @Success      200      {object}  dtos.RemoveFromCartResponse
// @Failure      400      {object}  dtos.ErrorResponse
// @Failure      500      {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart/remove/{cart_id} [delete]
func RemoveFromCartHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.RemoveFromCartRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Extract cart ID from path variables
	cartID := mux.Vars(r)["cart_id"]

	// Delete cart item from database
	if err := models.DeleteCartItem(models.DB, cartID, req.ProductID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to remove cart item with Cart ID " + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch updated cart items
	res, err := getCartItemsByCartID(cartID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: failedToGetCartItems + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Respond with updated cart items
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Product removed from cart successfully",
			Code:        http.StatusOK,
		},
		Payload:   res,
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
// func ApplyDiscountHandler(w http.ResponseWriter, r *http.Request) {
// 	start := time.Now()
// 	// Read and restore body FIRST
// 	requestSummary := utils.GetRequestSummary(r)
// 	req, ok := DecodeRequestBody[dtos.CouponRequest](r, w, requestSummary, start)
// 	if !ok {
// 		return
// 	}
// 	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Cart") {
// 		return
// 	}
// 	//validate coupon/promc code/voucher
// 	items, err := validateCodeVoucher(*req)
// 	if err != nil {
// 		log.Printf("Error fetching cart items: %v", err)
// 		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
// 			CollectiveInfo: utils.CollectiveInfo{
// 				Module:      "Cart",
// 				Description: "Failed to apply discount to order ID " + req.OrderID,
// 				Code:        http.StatusInternalServerError,
// 			},
// 			Message:   err.Error(),
// 			TimeTaken: time.Since(start),
// 			Function:  utils.GetCurrentFuncName(),
// 			Request:   r,
// 			RawBody:   requestSummary})
// 		return
// 	}
// 	//update order with new totals in the database
// 	if err := models.UpdateOrderTotals(items); err != nil {
// 		log.Printf("Error updating order totals: %v", err)
// 		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
// 			CollectiveInfo: utils.CollectiveInfo{
// 				Module:      "Cart",
// 				Description: "Failed to update order totals for Order ID " + req.OrderID,
// 				Code:        http.StatusInternalServerError,
// 			},
// 			Message:   err.Error(),
// 			TimeTaken: time.Since(start),
// 			Function:  utils.GetCurrentFuncName(),
// 			Request:   r,
// 			RawBody:   requestSummary})
// 		return
// 	}

//		utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
//			CollectiveInfo: utils.CollectiveInfo{
//				Module:      "Cart",
//				Description: "Discount applied successfully to Order ID " + req.OrderID,
//				Code:        http.StatusOK,
//			},
//			Payload:   items,
//			Message:   "Discount applied successfully",
//			TimeTaken: time.Since(start),
//			Function:  utils.GetCurrentFuncName(),
//			Request:   r,
//			RawBody:   requestSummary})
//	}
//
// ApplyDiscountHandler applies a discount coupon, voucher, or promo code to the cart.
//
// @Summary      Apply Discount
// @Description  Apply a discount coupon, voucher, or promo code to the user's cart.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        coupon  body      dtos.CouponRequest  true  "Discount Code Details"
// @Success      200     {object}  dtos.ViewCartResponse
// @Failure      400     {object}  dtos.ErrorResponse
// @Failure      500     {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart/apply-coupon [post]
func ApplyDiscountHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.CouponRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Determine discount type if not provided
	req.DiscountType = models.GetDiscountCodeType(models.DB, req.Code)
	if req.RequestType == "" {
		req.RequestType = "check"
	}

	// Validate the request payload
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Cart") {
		return
	}

	// Validate and apply the discount code
	items, err := validateCodeVoucher(*req)
	if err != nil {
		log.Printf("Error fetching cart items: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to apply discount to cart ID " + req.CartID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// // Reset totals if negative
	// if items.TotalAmount <= 0 {
	// 	items.SubTotal = 0
	// 	items.EstimatedTax = 0
	// }

	// Calculate delivery charge if location ID is provided
	if req.LocationID != nil {
		locationIDInt := *req.LocationID
		if locationIDInt != 0 {
			loc, err := models.GetLocationByID(models.DB, locationIDInt)
			if err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Cart",
						Description: "Failed to fetch location with ID " + strconv.Itoa(locationIDInt),
						Code:        http.StatusBadRequest,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary})
				return
			}
			items.DeliverCharge = loc.Charge
			items.TotalAmount += loc.Charge

		}
	}

	// Respond with updated cart details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Discount applied successfully to Cart ID " + req.CartID,
			Code:        http.StatusOK,
		},
		Payload:   items,
		Message:   "Discount applied successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func validateCodeVoucher(req dtos.CouponRequest) (dtos.ViewCartResponse, error) {
	cartData, err := getCartItemsByCartID(req.CartID)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}

	switch req.DiscountType {
	case "coupon":
		return applyCoupon(cartData, req.Code)
	case "voucher":
		return applyVoucher(cartData, req.Code, req.RequestType)
	case "promo_code":
		return applyPromoCode(cartData, req.Code, req.RequestType)
	default:
		return dtos.ViewCartResponse{}, errors.New("invalid promo type")
	}
}
func applyCoupon(cart dtos.ViewCartResponse, code string) (dtos.ViewCartResponse, error) {
	discount, err := models.ValidateCoupon(models.DB, code)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}

	if discount > cart.TotalAmount {
		discount = cart.TotalAmount
	}
	cart.Discount += discount
	cart.TotalAmount -= discount
	return cart, nil
}

// http handler to just validate the code without applying it, this is for the frontend to check if the code is valid before applying it
func ValidateVoucherCodeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Decode request body
	req, ok := DecodeRequestBody[dtos.VoucherCode](r, w, requestSummary, start)
	if !ok {
		return
	}
	// Validate the request payload
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Cart") {
		return
	}
	// Validate the voucher code
	voucherBalance, err := models.ValidateVoucher(models.DB, req.Code)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to validate voucher code: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// voucher balance should always be greater than amount passed from frontend, if not return error
	if voucherBalance < req.Amount {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Voucher balance is less than the amount to be applied",
				Code:        http.StatusBadRequest,
			},
			Message:   "Voucher balance is less than the amount to be applied",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Respond with voucher balance
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Voucher code validated successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"voucher_balance": voucherBalance},
		Message:   "Voucher code validated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func applyVoucher(cart dtos.ViewCartResponse, code string, requestType string) (dtos.ViewCartResponse, error) {
	voucherBalance, err := models.ValidateVoucher(models.DB, code)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}

	//check if voucher balance is greater than or equal to cart total amount, if yes then apply the voucher balance to the cart total amount and set the cart total amount to 0, if not then apply the voucher balance to the cart total amount and reduce the cart total amount by the voucher balance
	if voucherBalance < cart.TotalAmount {
		return dtos.ViewCartResponse{}, errors.New("voucher balance is less than the cart total amount")
	}

	var discount float64
	if voucherBalance >= cart.TotalAmount {
		discount = cart.TotalAmount
		cart.TotalAmount = 0
	} else {
		discount = voucherBalance
		cart.TotalAmount -= voucherBalance
	}
	cart.Discount += discount
	if requestType == "apply" {
		if err := models.UpdateVoucherBalance(models.DB, code, voucherBalance-discount); err != nil {
			return dtos.ViewCartResponse{}, err
		}
		//add cart history
		if err := models.AddVoucherHistory(models.DB, code, discount, cart.CartItems); err != nil {
			return dtos.ViewCartResponse{}, err
		}
	}
	return cart, nil
}

func applyPromoCode(cart dtos.ViewCartResponse, code string, requestType string) (dtos.ViewCartResponse, error) {
	log.Print("request type: ", requestType)
	promoData, err := models.ValidatePromoCode(models.DB, code, cart.TotalAmount)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}

	var discount float64

	promoType := strings.ToLower(*promoData.PromoType)

	if promoType == "brand" && promoData.BrandID != nil && *promoData.BrandID != "" {
		brandDiscount, err := calculateBrandDiscount(*promoData.BrandID, cart.CartItems, promoData)
		if err != nil {
			return dtos.ViewCartResponse{}, err
		}
		discount = brandDiscount
	} else {
		switch promoData.DiscountType {
		case "FIXED":
			discount = promoData.DiscountValue
			if discount > cart.TotalAmount {
				discount = cart.TotalAmount
			}
		case "PERCENTAGE":
			discount = (cart.TotalAmount * promoData.DiscountValue) / 100
			if discount > cart.TotalAmount {
				discount = cart.TotalAmount
			}
		default:
			return dtos.ViewCartResponse{}, fmt.Errorf("unsupported discount type")
		}
	}

	cart.Discount += discount
	cart.TotalAmount -= discount
	//update promo code usage count
	if requestType == "apply" {
		if err := models.IncrementPromoCodeUsage(models.DB, code); err != nil {
			return dtos.ViewCartResponse{}, err
		}
	}
	return cart, nil
}

// calculateBrandDiscount calculates discount for products in a specific brand
func calculateBrandDiscount(brandID string, items []dtos.CartItem, promoData dtos.PromoCodeData) (float64, error) {
	var totalBrandItemsAmount float64

	// Iterate through cart items and check if the product belongs to the brand
	for _, item := range items {
		isBrandProduct, err := models.IsProductInBrand(models.DB, item.Product.ID, brandID)
		if err != nil {
			log.Printf("Error checking product brand: %v", err)
			continue
		}

		if isBrandProduct {
			// Calculate the total amount for this specific item in the cart
			itemTotal := float64(item.Quantity) * item.Product.Price
			totalBrandItemsAmount += itemTotal
		}
	}

	if totalBrandItemsAmount == 0 {
		return 0, fmt.Errorf("no products from the specified brand in the cart")
	}

	// Calculate discount based on totalBrandItemsAmount
	var discount float64
	switch promoData.DiscountType {
	case "FIXED":
		discount = promoData.DiscountValue
		if discount > totalBrandItemsAmount {
			discount = totalBrandItemsAmount
		}
	case "PERCENTAGE":
		discount = (totalBrandItemsAmount * promoData.DiscountValue) / 100
		if discount > totalBrandItemsAmount {
			discount = totalBrandItemsAmount
		}
	default:
		return 0, fmt.Errorf("unsupported discount type")
	}

	return discount, nil
}
func applyPromoCodeToOrder(db models.DBExecutor, totalAmount, totalDiscount float64, code string, promoCodeType string, order dtos.OrderRequest) (float64, float64, error) {
	switch promoCodeType {
	case "promo_code":
		return applyPromoCodeDiscount(db, totalAmount, totalDiscount, code, order)
	case "coupon":
		return applyCouponDiscount(db, totalAmount, totalDiscount, code)
	case "voucher":
		return applyVoucherDiscount(db, totalAmount, totalDiscount, code)
	default:
		return totalAmount, totalDiscount, fmt.Errorf("invalid discount type for order")
	}
}

func applyPromoCodeDiscount(db models.DBExecutor, totalAmount, totalDiscount float64, code string, order dtos.OrderRequest) (float64, float64, error) {
	promoData, err := models.ValidatePromoCode(db, code, totalAmount)
	if err != nil {
		return 0, 0, err
	}

	discount := calculatePromoDiscount(promoData, totalAmount, order)
	if discount < 0 {
		return 0, 0, fmt.Errorf("unsupported discount type")
	}

	totalDiscount += discount
	if err := models.IncrementPromoCodeUsage(db, code); err != nil {
		return 0, 0, err
	}

	return totalAmount, totalDiscount, nil
}

func calculatePromoDiscount(promoData dtos.PromoCodeData, totalAmount float64, order dtos.OrderRequest) float64 {
	var discount float64
	promoType := strings.ToLower(*promoData.PromoType)

	if promoType == "brand" && promoData.BrandID != nil && *promoData.BrandID != "" {
		var items []dtos.CartItem
		for _, orderItem := range order.OrderItems {
			items = append(items, dtos.CartItem{
				Quantity: orderItem.Quantity,
				Product: dtos.Product{
					ID:    orderItem.ProductID,
					Price: orderItem.UnitPrice,
				},
			})
		}
		brandDiscount, err := calculateBrandDiscount(*promoData.BrandID, items, promoData)
		if err != nil {
			return 0
		}
		discount = brandDiscount
	} else {
		switch promoData.DiscountType {
		case "FIXED":
			discount = promoData.DiscountValue
		case "PERCENTAGE":
			discount = (totalAmount * promoData.DiscountValue) / 100
		default:
			return -1
		}

		if discount > totalAmount {
			discount = totalAmount
		}
	}
	return discount
}

func applyCouponDiscount(db models.DBExecutor, totalAmount, totalDiscount float64, code string) (float64, float64, error) {
	discount, err := models.ValidateCoupon(db, code)
	if err != nil {
		return 0, 0, err
	}
	if discount > totalAmount {
		discount = totalAmount
	}
	totalDiscount += discount
	totalAmount -= discount
	return totalAmount, totalDiscount, nil
}

func applyVoucherDiscount(db models.DBExecutor, totalAmount, totalDiscount float64, code string) (float64, float64, error) {
	voucherBalance, err := models.ValidateVoucher(db, code)
	if err != nil {
		return 0, 0, err
	}
	if voucherBalance > totalAmount {
		voucherBalance = totalAmount
	}
	totalDiscount += voucherBalance
	totalAmount -= voucherBalance
	return totalAmount, totalDiscount, nil
}
