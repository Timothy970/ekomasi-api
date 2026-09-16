package handlers

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
)

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
