package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

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
// func ApplyDiscountHandler(c *gin.Context) {
// 	start := time.Now()
// 	// Read and restore body FIRST
// 	requestSummary := utils.GetRequestSummary(c.Request)
// 	req, ok := DecodeRequestBody[dtos.CouponRequest](c, requestSummary, start)
// 	if !ok {
// 		return
// 	}
// 	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Cart") {
// 		return
// 	}
// 	//validate coupon/promc code/voucher
// 	items, err := validateCodeVoucher(*req)
// 	if err != nil {
// 		log.Printf("Error fetching cart items: %v", err)
// 		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
// 			CollectiveInfo: utils.CollectiveInfo{
// 				Module:      "Cart",
// 				Description: "Failed to apply discount to order ID " + req.OrderID,
// 				Code:        http.StatusInternalServerError,
// 			},
// 			Message:   err.Error(),
// 			TimeTaken: time.Since(start),
// 			Function:  utils.GetCurrentFuncName(),
// 			Request: c.Request,
// 			RawBody:   requestSummary})
// 		return
// 	}
// 	//update order with new totals in the database
// 	if err := models.UpdateOrderTotals(items); err != nil {
// 		log.Printf("Error updating order totals: %v", err)
// 		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
// 			CollectiveInfo: utils.CollectiveInfo{
// 				Module:      "Cart",
// 				Description: "Failed to update order totals for Order ID " + req.OrderID,
// 				Code:        http.StatusInternalServerError,
// 			},
// 			Message:   err.Error(),
// 			TimeTaken: time.Since(start),
// 			Function:  utils.GetCurrentFuncName(),
// 			Request: c.Request,
// 			RawBody:   requestSummary})
// 		return
// 	}

//		utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
//			CollectiveInfo: utils.CollectiveInfo{
//				Module:      "Cart",
//				Description: "Discount applied successfully to Order ID " + req.OrderID,
//				Code:        http.StatusOK,
//			},
//			Payload:   items,
//			Message:   "Discount applied successfully",
//			TimeTaken: time.Since(start),
//			Function:  utils.GetCurrentFuncName(),
//			Request: c.Request,
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
func ApplyDiscountHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.CouponRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Determine discount type if not provided
	req.DiscountType = models.GetDiscountCodeType(models.DB, req.Code)
	if req.RequestType == "" {
		req.RequestType = "check"
	}

	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Cart") {
		return
	}

	// Validate and apply the discount code
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	items, err := validateCodeVoucher(*req, tenantID)
	if err != nil {
		log.Printf("Error fetching cart items: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to apply discount to cart ID " + req.CartID,
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
				utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Cart",
						Description: "Failed to fetch location with ID " + strconv.Itoa(locationIDInt),
						Code:        http.StatusBadRequest,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   c.Request,
					RawBody:   requestSummary})
				return
			}
			items.DeliverCharge = loc.Charge
			items.TotalAmount += loc.Charge

		}
	}

	// Respond with updated cart details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Discount applied successfully to Cart ID " + req.CartID,
			Code:        http.StatusOK,
		},
		Payload:   items,
		Message:   "Discount applied successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
func validateCodeVoucher(req dtos.CouponRequest, tenantID int) (dtos.ViewCartResponse, error) {
	cartData, err := getCartItemsByCartID(req.CartID, tenantID)
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
func ValidateVoucherCodeHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Decode request body
	req, ok := DecodeRequestBody[dtos.VoucherCode](c, requestSummary, start)
	if !ok {
		return
	}
	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Cart") {
		return
	}
	// Validate the voucher code
	voucherBalance, err := models.ValidateVoucher(models.DB, req.Code)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to validate voucher code: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// voucher balance should always be greater than amount passed from frontend, if not return error
	if voucherBalance < req.Amount {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Voucher balance is less than the amount to be applied",
				Code:        http.StatusBadRequest,
			},
			Message:   "Voucher balance is less than the amount to be applied",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Respond with voucher balance
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Voucher code validated successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"voucher_balance": voucherBalance},
		Message:   "Voucher code validated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
