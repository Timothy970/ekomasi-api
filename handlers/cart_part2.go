package handlers

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

func getCartItemsByCartID(cartID string, tenantID int) (dtos.ViewCartResponse, error) {
	items, err := models.GetCartItems(models.DB, cartID, tenantID)
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
func UpdateCartItemHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract cart ID from path variables
	cartID := c.Param("cart_id")

	// Decode request body
	req, ok := DecodeRequestBody[dtos.UpdateCartItemRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Update cart item in database
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	if err := models.UpdateCartItem(models.DB, cartID, req.ProductID, req.Quantity, tenantID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to update cart item with Cart ID " + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch updated cart items
	res, err := getCartItemsByCartID(cartID, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: failedToGetCartItems + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with updated cart items
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Cart item updated successfully with Cart ID " + cartID,
			Code:        http.StatusOK,
		},
		Payload:   res,
		Message:   "Cart item updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func RemoveFromCartHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.RemoveFromCartRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Extract cart ID from path variables
	cartID := c.Param("cart_id")

	// Delete cart item from database
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	if err := models.DeleteCartItem(models.DB, cartID, req.ProductID, tenantID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to remove cart item with Cart ID " + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch updated cart items
	res, err := getCartItemsByCartID(cartID, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: failedToGetCartItems + cartID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with updated cart items
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Product removed from cart successfully",
			Code:        http.StatusOK,
		},
		Payload:   res,
		Message:   "Product removed from cart successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
