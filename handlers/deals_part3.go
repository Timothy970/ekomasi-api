package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func parseDealProductRequest(r *http.Request) (*dtos.FlashDealProducts, error) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, fmt.Errorf("image is required")
	}
	defer file.Close()

	url := "https://example.com/image.jpg" // Placeholder URL since GCS upload is not implemented here

	dealType := r.FormValue("deal_type")
	brandID := r.FormValue("brand_id")
	if dealType == "" {
		dealType = "product"
	}
	brandDiscount := r.FormValue("discount")
	brandDiscountType := r.FormValue("discount_type")
	if dealType == "brand" && (brandID == "" || brandDiscount == "" || brandDiscountType == "") {
		return nil, fmt.Errorf("brand ID, discount, and discount type are required for brand deals")
	}
	products, err := parseProducts(r.FormValue("products"), brandID, dealType, brandDiscount, brandDiscountType)
	if err != nil {
		return nil, err
	}
	return &dtos.FlashDealProducts{
		Title:    r.FormValue("title"),
		Image:    url,
		Duration: r.FormValue("duration"),
		Products: products,
		DealType: dealType,
		BrandID:  &brandID,
	}, nil
}
func parseProducts(productsStr string, brandID, dealType, brandDiscount, brandDiscountType string) ([]dtos.ProductsDeal, error) {
	var products []dtos.ProductsDeal
	if dealType == "product" {
		if productsStr == "" {
			return products, nil
		}

		if err := json.Unmarshal([]byte(productsStr), &products); err != nil {
			return nil, fmt.Errorf("invalid products format: %w", err)
		}
		if len(products) == 0 {
			return products, fmt.Errorf("products array cannot be empty for product deals")
		}
	} else if dealType == "brand" {
		var err error
		products, err = models.GetProductIDsByBrandID(models.DB, brandID, brandDiscount, brandDiscountType)
		if err != nil {
			return nil, fmt.Errorf("failed to get products by brand ID: %w", err)
		}
	} else {
		return nil, fmt.Errorf("invalid deal type: %s", dealType)
	}
	return products, nil
}
func validateProductsExist(products []dtos.ProductsDeal, c *gin.Context, start time.Time, requestSummary string) error {
	for _, p := range products {
		err := models.IsProductThere(models.DB, p.ProductID)
		if err != nil {
			if err.Error() == "product not found" {
				err = fmt.Errorf("product with ID %s not found", p.ProductID)
			}
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Deals",
					Description: "Failed to validate product existence when creating deal",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return err
		}
	}
	return nil
}
func parseDuration(duration string) (time.Time, time.Time, error) {
	var startStr, endStr string
	if _, err := fmt.Sscanf(duration, "%s to %s", &startStr, &endStr); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format. expected 'YYYY-MM-DD to YYYY-MM-DD'")
	}
	return models.StringToTime(startStr), models.StringToTime(endStr), nil
}
func createDeal(req *dtos.FlashDealProducts, startDate, endDate time.Time) (string, error) {
	dealData := dtos.CreateDeal{
		Name:      req.Title,
		StartDate: startDate,
		EndDate:   endDate,
		Image:     req.Image,
		DealType:  req.DealType,
		BrandID:   req.BrandID,
	}
	return models.CreateDeal(models.DB, dealData)
}

func addProductsToDeal(dealID string, products []dtos.ProductsDeal, c *gin.Context, start time.Time, requestSummary string) error {
	for _, p := range products {
		discount := float64(p.Discount)
		if err := models.AddProductToDeal(models.DB, dealID, p.ProductID, &p.DiscountType, &discount); err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Deals",
					Description: "Failed to add product to deal when creating deal with deal ID " + dealID,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return err
		}
	}
	return nil
}
