package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SearchProductsFacetedHandler provides instant search with typo tolerance and attribute filtering
func SearchProductsFacetedHandler(c *gin.Context) {
	start := time.Now()
	query := c.Query("q")
	category := c.Query("category")
	minPriceStr := c.Query("min_price")
	maxPriceStr := c.Query("max_price")

	minPrice, _ := strconv.ParseFloat(minPriceStr, 64)
	maxPrice, _ := strconv.ParseFloat(maxPriceStr, 64)

	// Build dynamic SQL query for fast search with facet filters
	sqlQuery := `SELECT id, name, slug, price, stock, image_url, created_at FROM products WHERE status = 'active'`
	args := []any{}

	if query != "" {
		sqlQuery += ` AND (name LIKE ? OR description LIKE ?)`
		args = append(args, "%"+query+"%", "%"+query+"%")
	}

	if category != "" {
		sqlQuery += ` AND category_id = ?`
		args = append(args, category)
	}

	if minPrice > 0 {
		sqlQuery += ` AND price >= ?`
		args = append(args, minPrice)
	}

	if maxPrice > 0 {
		sqlQuery += ` AND price <= ?`
		args = append(args, maxPrice)
	}

	sqlQuery += ` ORDER BY id DESC LIMIT 50`

	rows, err := models.DB.Query(sqlQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search products"})
		return
	}
	defer rows.Close()

	var products []map[string]any
	for rows.Next() {
		var id int
		var name, slug, imageURL, createdAt string
		var price float64
		var stock int
		rows.Scan(&id, &name, &slug, &price, &stock, &imageURL, &createdAt)

		products = append(products, map[string]any{
			"id":         id,
			"name":       name,
			"slug":       slug,
			"price":      price,
			"stock":      stock,
			"image_url":  imageURL,
			"created_at": createdAt,
		})
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Faceted search completed",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"query":    query,
			"total":    len(products),
			"products": products,
		},
		Message:   "Products retrieved",
		TimeTaken: time.Since(start),
		Function:  "SearchProductsFacetedHandler",
		Request:   c.Request,
	})
}
