package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"fmt"

	"github.com/teris-io/shortid"
)

func fetchProductDiscount(db DBExecutor, productID string) (dtos.DiscountType, error) {
	query := `
		SELECT 
			pt.id, 
			pt.name, 
			pt.description, 
			pt.value
		FROM promotion_types pt
		JOIN product_discounts pd 
			ON pt.id = pd.promotion_type_id
		WHERE pd.product_id = ?
	`
	var discount dtos.DiscountType
	err := db.QueryRow(query, productID).Scan(&discount.ID, &discount.Name, &discount.Description, &discount.Value)
	if err != nil {
		if err == sql.ErrNoRows {
			return dtos.DiscountType{}, nil // No discount is not an error, return empty struct
		}
		return dtos.DiscountType{}, err
	}
	return discount, nil
}

// getBundleProducts fetches all products that are part of a bundle.
//
// This function retrieves products linked to a bundle product through the bundle_products table.
// Each product is enriched with complete details including images, warranties, features, variants, and tax.
//
// Parameters:
//   - bundleID: string - The product_id of the bundle (from bundle_products.bundle_id)
//
// Returns:
//   - []dtos.Product: Array of products in the bundle with complete details
//   - error: Database error or nil on success
func getBundleProducts(db DBExecutor, bundleID string) ([]dtos.Product, error) {
	// Query to get all products in the bundle
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at, 
			c.name, p.tag, p.details, dp.discount, dp.discount_type, 
			ps.weight, ps.dimensions, ps.manufacturer, ps.weight_limit, bp.quantity
		FROM bundle_products bp
		INNER JOIN products p ON bp.product_id = p.product_id
		LEFT JOIN categories c ON p.category_id = c.category_id
		LEFT JOIN deal_products dp ON p.product_id = dp.product_id
		LEFT JOIN product_specifications ps ON p.product_id = ps.product_id
		WHERE bp.bundle_id = ?
	`

	rows, err := db.Query(query, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product

	for rows.Next() {
		var (
			p            dtos.Product
			detailsData  []byte
			categoryID   sql.NullString
			categoryName sql.NullString
		)

		// Scan product data
		err = rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &categoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
			&categoryName, &p.Tag, &detailsData, &p.Discount, &p.DiscountType,
			&p.Weight, &p.Dimensions, &p.Manufacturer, &p.WeightLimit, &p.BundleQuantity,
		)
		if err != nil {
			return nil, err
		}

		if categoryID.Valid {
			p.CategoryID = categoryID.String
		}
		if categoryName.Valid {
			p.CategoryName = categoryName.String
		}

		// Unmarshal JSON details if present
		if len(detailsData) > 0 {
			err := json.Unmarshal(detailsData, &p.Details)
			if err != nil {
				return nil, err
			}
		} else {
			p.Details = []string{}
		}

		// Fetch associated product images
		images, err := fetchProductImages(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Images = images

		// Fetch product warranties
		warranties, err := FetchProductWarranties(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Warranty = &warranties

		// Fetch product features
		features, err := fetchProductFeatures(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Features = features

		// Fetch product variants
		variants, err := getProductVariants(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.ProductVariants = variants
		p.VariantSelection, err = GetVariantSelection(db, p.ID)
		if err != nil {
			return nil, err
		}
		// Fetch tax information
		// tax, err := fetchProductTax(db, p.ID)
		// if err != nil {
		// 	return nil, err
		// }
		// p.Tax = &tax

		p.StockQuantity = p.BundleQuantity

		products = append(products, p)
	}

	return products, nil
}

// IsSkuThere validates that a SKU does not already exist in the database.
//
// This function is used before product creation to ensure SKU uniqueness.
//
// Parameters:
//   - sku: string - The SKU (Stock Keeping Unit) to check
//
// Returns:
//   - error: "duplicate SKU found: <sku>" if SKU exists, database error, or nil if unique
func IsSkuThere(db DBExecutor, sku string) error {
	// Check if SKU exists in products table
	skuExists, err := RecordExists(db, "products", "sku = ?", sku)
	if err != nil {
		return err
	}
	if skuExists {
		return fmt.Errorf("duplicate SKU found: %s", sku)
	}
	return nil
}

// IsCategoryParent validates that a category is not a parent category.
//
// This function prevents adding products to parent categories, enforcing
// that products must be added to leaf (subcategory) nodes only.
//
// Parameters:
//   - categoryID: string - The category_id to validate
//
// Returns:
//   - error: "cannot add product to a parent category..." if category is a parent,
//     database error, or nil if category is a valid subcategory
func IsCategoryParent(db DBExecutor, categoryID string) error {
	var parentID *string

	// Check if category has a parent (null parent_category_id = root/parent category)
	err := db.QueryRow("SELECT parent_category_id FROM categories WHERE category_id = ?", categoryID).Scan(&parentID)
	if err != nil {
		return err
	}

	// Prevent adding product to parent category (parentID is null)
	if parentID == nil {
		return fmt.Errorf("cannot add product to a parent category with ID %s, choose a subcategory instead", categoryID)
	}
	return nil

}

// AddNewProduct creates a new product with validation and default values.
//
// This function validates SKU uniqueness, category existence, and ensures products
// are only added to subcategories (not parent categories). It handles optional fields
// and marshals product details to JSON.
//
// Parameters:
//   - input: dtos.CreateProduct - Product details containing:
//   - Name, Description, SKU, Price (required)
//   - CategoryID: Must be a valid subcategory (not parent)
//   - StockQuantity, SearchVector, Tag
//   - LowStockAlert: Low stock warning threshold
//   - SellWhenOOS: Pointer to bool (allow selling when out of stock)
//   - ShowStock: Pointer to bool (show stock quantity to customers)
//   - BuyingPrice: Cost price
//   - Details: Array of strings (marshaled to JSON)
//   - userID: string - ID of the user creating the product (for audit trail)
//
// Returns:
//   - *dtos.CreateProduct: Created product with generated ID
//   - error: "duplicate SKU", category validation error, database error, or nil on success
func AddNewProduct(db DBExecutor, input dtos.CreateProduct, userID string, tenantID int) (*dtos.CreateProduct, error) {
	// Validate SKU uniqueness
	skuExists, err := RecordExists(db, "products", "sku = ?", input.SKU)
	if err != nil {
		return nil, err
	}
	if skuExists {
		return nil, fmt.Errorf("duplicate SKU")
	}

	// Validate category exists
	err = CategoryExists(db, input.CategoryID)
	if err != nil {
		return nil, err
	}

	// Ensure product is added to subcategory, not parent category
	err = IsCategoryParent(db, input.CategoryID)
	if err != nil {
		return nil, err
	}

	// Generate unique product ID
	productID, _ := shortid.Generate()

	// Marshal product details to JSON
	var detailsJSON []byte
	if input.Details != nil {
		detailsJSON, err = json.Marshal(input.Details)
		if err != nil {
			return nil, err
		}
	} else {
		detailsJSON = nil
	}

	// Insert product record
	//The price is set to 0 by default and will be updated later when
	_, err = db.Exec(`
		INSERT INTO products (product_id, name, description, sku, price, category_id, stock_quantity, search_vector, tag, low_stock_quantity_warning, created_by_id, buying_price, details, barcode, tenant_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		productID, input.Name, input.Description, input.SKU, 0, input.CategoryID, input.StockQuantity, input.SearchVector, input.Tag, input.LowStockAlert, userID, input.BuyingPrice, detailsJSON, input.Barcode, tenantID,
	)
	if err != nil {
		return nil, err
	}

	// Return created product summary
	return &dtos.CreateProduct{
		ID:            productID,
		Name:          input.Name,
		Description:   input.Description,
		SKU:           input.SKU,
		Price:         input.Price,
		CategoryID:    input.CategoryID,
		StockQuantity: input.StockQuantity,
		SearchVector:  input.SearchVector,
		Tag:           input.Tag,
		Details:       input.Details,
		Barcode:       input.Barcode,
	}, nil
}

// IsProductInBrand checks if a product belongs to a specific brand.
// Under product_variants table the variant_id is the brand_id and there is product_id.
//
// Parameters:
//   - db: DBExecutor
//   - productID: The product ID to check
//   - brandID: The brand ID (variant_id) to check against
//
// Returns:
//   - bool: true if the product belongs to the brand
//   - error: Database error or nil on success
func IsProductInBrand(db DBExecutor, productID string, brandID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM product_variants 
			WHERE product_id = ? AND variant_id = ?
		)
	`
	err := db.QueryRow(query, productID, brandID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// UpdateProductByID updates an existing product with SKU uniqueness validation.
//
// This function updates product details while ensuring the SKU remains unique
// (excluding the current product). It auto-updates the last_updated_at timestamp.
//
// Parameters:
//   - productID: string - The product_id to update
//   - input: dtos.CreateProduct - Updated product details:
//   - Name, Description, SKU, Price
//   - StockQuantity, SearchVector, Tag
//   - LowStockAlert: Low stock warning threshold
//   - SellWhenOOS: Allow selling when out of stock
//   - ShowStock: Show stock quantity to customers
//
// Returns:
//   - *dtos.CreateProduct: Updated product summary
//   - error: "product not found", "duplicate SKU", database error, or nil on success
