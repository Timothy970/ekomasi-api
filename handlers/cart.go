package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	// if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
	// 	return
	// }

	cartID, err := models.CreateCart(*req)
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
	cartID, err := models.GetUserCart(user.ID)
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

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Cart") {
		return
	}
	if err := models.InsertCartItem(req.CartID, req.ProductID, req.Quantity); err != nil {
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
	locationID := r.URL.Query().Get("location_id")
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
	if locationID != "" {
		locationIDInt, _ := strconv.Atoi(locationID)
		if locationIDInt != 0 {
			loc, err := models.GetLocationByID(locationIDInt)
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
			res.Final += loc.Charge

		}
	}
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
	items, err := models.GetCartItems(cartID)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}
	discount := 0.0
	total := 0.0
	for _, item := range items {
		//check if any product has a discount
		productDiscount, err := models.GetProductPromotionData(item.Product.ID)
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
		total += float64(item.Quantity) * item.Product.Price
	}
	//get estimated tax
	estimatedTax, err := models.GetEstimatedTax()
	estimatedTaxValue := (estimatedTax * total) / 100
	res := dtos.ViewCartResponse{
		CartItems:    items,
		Total:        total - discount - estimatedTaxValue,
		Final:        total,
		Discount:     discount,
		EstimatedTax: estimatedTaxValue,
	}
	return res, nil
}
func getOrderItemsByOrderID(orderID string) (*dtos.Order, error) {
	items, err := models.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}
	return items, nil
}
func calculateDifferentDiscountTypes(promo *dtos.PromotionData, item dtos.CartItem) (float64, error) {
	switch promo.Type {
	case "Percentage":
		// Apply a percentage discount to total item price
		return (promo.Value / 100) * float64(item.Quantity) * item.Product.Price, nil

	case "Fixed":
		// Apply a fixed amount discount (flat rate)
		// Split proportionally if needed; here we apply it fully if item subtotal > discount
		subtotal := float64(item.Quantity) * item.Product.Price
		if subtotal > promo.Value {
			return promo.Value, nil
		}
		return subtotal, nil

	case "BOGO":
		// Buy One Get One Free
		// For every two items, one is free
		if item.Quantity > 1 {
			freeItems := item.Quantity / 2
			return float64(freeItems) * item.Product.Price, nil
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
func ApplyDiscountHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.CouponRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Cart") {
		return
	}
	//validate coupon/promc code/voucher
	items, err := validateCodeVoucher(*req)
	if err != nil {
		log.Printf("Error fetching cart items: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to apply discount to cart ID " + req.CartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if req.LocationID != nil {
		locationIDInt := *req.LocationID
		if locationIDInt != 0 {
			loc, err := models.GetLocationByID(locationIDInt)
			if err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Cart",
						Description: "Failed to fetch location with ID " + strconv.Itoa(locationIDInt),
						Code:        http.StatusInternalServerError,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary})
				return
			}
			items.DeliverCharge = loc.Charge
			items.Final += loc.Charge

		}
	}

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
		return applyVoucher(cartData, req.Code)
	case "promo_code":
		return applyPromoCode(cartData, req.Code)
	default:
		return dtos.ViewCartResponse{}, errors.New("invalid promo type")
	}
}
func applyCoupon(cart dtos.ViewCartResponse, code string) (dtos.ViewCartResponse, error) {
	discount, err := models.ValidateCoupon(code)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}

	if discount > cart.Total {
		discount = cart.Total
	}
	cart.Discount += discount
	cart.Total -= discount
	return cart, nil
}

func applyVoucher(cart dtos.ViewCartResponse, code string) (dtos.ViewCartResponse, error) {
	voucherBalance, err := models.ValidateVoucher(code)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}

	var discount float64
	if voucherBalance >= cart.Total {
		discount = cart.Total
		cart.Total = 0
	} else {
		discount = voucherBalance
		cart.Total -= voucherBalance
	}
	cart.Discount += discount
	// if requestType == "apply" {
	// 	if err := models.UpdateVoucherBalance(code, voucherBalance-discount); err != nil {
	// 		return dtos.ViewCartResponse{}, err
	// 	}
	// 	//add cart history
	// 	if err := models.AddVoucherHistory(code, discount, cart.Items); err != nil {
	// 		return dtos.ViewCartResponse{}, err
	// 	}
	// }
	return cart, nil
}

func applyPromoCode(cart dtos.ViewCartResponse, code string) (dtos.ViewCartResponse, error) {
	promoData, err := models.ValidatePromoCode(code, cart.Total)
	if err != nil {
		return dtos.ViewCartResponse{}, err
	}

	var discount float64
	switch promoData.DiscountType {
	case "FIXED":
		discount = promoData.DiscountValue
		if discount > cart.Total {
			discount = cart.Total
		}
	case "PERCENTAGE":
		discount = (cart.Total * promoData.DiscountValue) / 100
		if discount > cart.Total {
			discount = cart.Total
		}
	default:
		return dtos.ViewCartResponse{}, fmt.Errorf("unsupported discount type")
	}

	cart.Discount += discount
	cart.Total -= discount
	// //update promo code usage count
	// if requestType == "apply" {
	// 	if err := models.IncrementPromoCodeUsage(code); err != nil {
	// 		return dtos.ViewCartResponse{}, err
	// 	}
	// }
	return cart, nil
}
func applyPromoCodeToOrder(totalAmount, totalDiscount float64, code string) (float64, float64, error) {
	promoData, err := models.ValidatePromoCode(code, totalAmount)
	if err != nil {
		return 0, 0, err
	}

	var discount float64
	switch promoData.DiscountType {
	case "FIXED":
		discount = promoData.DiscountValue
		if discount > totalAmount {
			discount = totalAmount
		}
	case "PERCENTAGE":
		discount = (totalAmount * promoData.DiscountValue) / 100
		if discount > totalAmount {
			discount = totalAmount
		}
	default:
		return 0, 0, fmt.Errorf("unsupported discount type")
	}

	totalDiscount += discount
	//update promo code usage count
	if err := models.IncrementPromoCodeUsage(code); err != nil {
		return 0, 0, err
	}

	return totalAmount, totalDiscount, nil
}
