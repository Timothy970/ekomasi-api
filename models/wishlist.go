package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"

	"github.com/teris-io/shortid"
)

var nowishlist = "Wishlist not found"
var fetchwishlist = "wishlist_id = ?"

func CreateWishList(body dtos.CreateWishlist, userID string) (*dtos.Wishlist, error) {
	wishlistID, _ := shortid.Generate()

	query := `INSERT INTO wishlists (wishlist_id, user_id, name, is_public) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, wishlistID, userID, body.Name, body.IsPublic)
	if err != nil {
		return nil, err
	}

	return &dtos.Wishlist{WishlistID: wishlistID, Name: body.Name, IsPublic: body.IsPublic}, nil
}

func GetAllUserWishList(userID, wishlistID string, limit, page int) ([]dtos.AllWishlist, *dtos.PaginationMeta, error) {
	if userID == "" {
		return nil, nil, errors.New("userID is required")
	}

	query, args := buildWishlistQuery(userID, wishlistID, limit, page)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var lists []dtos.AllWishlist
	for rows.Next() {
		wishlist, err := scanWishlistRow(rows)
		if err != nil {
			return nil, nil, err
		}

		products, err := fetchProductsForWishlist(wishlist.WishlistID)
		if err != nil {
			return nil, nil, err
		}

		wishlist.Products = products
		lists = append(lists, wishlist)
	}
	if wishlistID != "" && len(lists) == 0 {
		return nil, nil, fmt.Errorf("no wishlist found with ID: %s", wishlistID)
	}

	if wishlistID != "" {
		return lists, nil, nil
	}

	meta, err := buildPaginationMeta(userID, limit, page)
	if err != nil {
		return nil, nil, err
	}

	return lists, meta, nil
}
func buildWishlistQuery(userID, wishlistID string, limit, page int) (string, []interface{}) {
	query := `SELECT wishlist_id, name, is_public FROM wishlists WHERE user_id = ?`
	args := []interface{}{userID}

	if wishlistID != "" {
		query += ` AND wishlist_id = ?`
		args = append(args, wishlistID)
	} else {
		offset := (page - 1) * limit
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	return query, args
}

func scanWishlistRow(rows *sql.Rows) (dtos.AllWishlist, error) {
	var w dtos.AllWishlist
	err := rows.Scan(&w.WishlistID, &w.Name, &w.IsPublic)
	return w, err
}

func fetchProductsForWishlist(wishlistID string) ([]dtos.Product, error) {
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
		FROM wishlist_items wi
		JOIN products p ON wi.product_id = p.product_id
		WHERE wi.wishlist_id = ?
	`

	rows, err := DB.Query(query, wishlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dtos.Product
	for rows.Next() {
		var p dtos.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
			&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
		); err != nil {
			return nil, err
		}

		images, err := fetchProductImages(p.ID)
		if err != nil {
			return nil, err
		}
		p.Images = images
		products = append(products, p)
	}
	return products, nil
}

func buildPaginationMeta(userID string, limit, page int) (*dtos.PaginationMeta, error) {
	var totalItems int
	err := DB.QueryRow(`SELECT COUNT(*) FROM wishlists WHERE user_id = ?`, userID).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	totalPages := (totalItems + limit - 1) / limit
	return &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}, nil
}
func GetOrCreateWishlist(userID, wishlistName string) (string, error) {
	// Check if wishlist exists
	exists, err := RecordExists("wishlists", "user_id = ?", userID)
	if err != nil {
		return "", err
	}

	if exists {
		// Return existing wishlist ID
		return GetWishlistByUserID(userID)
	}

	// Create new wishlist
	return CreateNewWishList(wishlistName, userID)
}

func GetWishlistByUserID(userID string) (string, error) {
	var wishlistID string
	query := `SELECT wishlist_id FROM wishlists WHERE user_id = ? LIMIT 1`
	err := DB.QueryRow(query, userID).Scan(&wishlistID)
	if err != nil {
		return "", err
	}
	return wishlistID, nil
}
func CreateNewWishList(Name, userID string) (string, error) {
	wishlistID, _ := shortid.Generate()

	query := `INSERT INTO wishlists (wishlist_id, user_id, name, is_public) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, wishlistID, userID, Name, 1)
	if err != nil {
		return "", err
	}

	return wishlistID, nil
}
func CreateWishListItem(wishlistID, productID, userID string) error {
	//check if product exists
	err := isProductThere(productID)
	if err != nil {
		return err
	}

	//check is wishlist exists
	exists, err := RecordExists("wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}
	//check if item already in wishlist
	exists, err = RecordExists("wishlist_items", "wishlist_id = ? AND product_id = ?", wishlistID, productID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("product already in wishlist")
	}

	itemID, _ := shortid.Generate()

	query := `INSERT INTO wishlist_items (wishlist_item_id, wishlist_id, product_id) VALUES (?, ?, ?)`
	_, err = DB.Exec(query, itemID, wishlistID, productID)
	if err != nil {
		return err
	}
	return nil
	// return GetAllUserWishList(userID, "",0,)
}

func RemoveWishlistItem(wishlistID, productID, userID string) error {
	//check if product exists
	exists, err := RecordExists("products", "product_id = ?", productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not found")
	}
	//check is wishlist exists
	exists, err = RecordExists("wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}
	query := `DELETE FROM wishlist_items WHERE product_id = ?`
	_, err = DB.Exec(query, productID)
	if err != nil {
		return err
	}
	return nil
	// return GetAllUserWishList(userID)
}
func GetWishlistByID(wishlistID string) ([]dtos.AllWishlist, error) {
	// Step 1: Get all wishlists for the user
	query := `SELECT wishlist_id, name, is_public FROM wishlists WHERE wishlist_id = ?`
	rows, err := DB.Query(query, wishlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []dtos.AllWishlist
	for rows.Next() {
		var w dtos.AllWishlist
		if err := rows.Scan(&w.WishlistID, &w.Name, &w.IsPublic); err != nil {
			return nil, err
		}

		// Step 2: For each wishlist, fetch associated products
		productQuery := `
			SELECT 
				p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
				p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
			FROM wishlist_items wi
			JOIN products p ON wi.product_id = p.product_id
			WHERE wi.wishlist_id = ?
		`
		productRows, err := DB.Query(productQuery, w.WishlistID)
		if err != nil {
			return nil, err
		}

		var products []dtos.Product
		for productRows.Next() {
			var p dtos.Product
			if err := productRows.Scan(
				&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
				&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
			); err != nil {
				productRows.Close()
				return nil, err
			}
			products = append(products, p)
		}
		productRows.Close()

		w.Products = products
		lists = append(lists, w)
	}

	if len(lists) == 0 {
		return nil, nil
	}
	return lists, nil
}

// Delete a wishlist
func DeleteWishList(wishlistID, userID string) error {
	// Check if wishlist exists
	exists, err := RecordExists("wishlists", fetchwishlist, wishlistID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s", nowishlist)
	}
	_, err = DB.Exec("DELETE FROM wishlists WHERE wishlist_id = ? AND user_id = ?", wishlistID, userID)
	return err
}

func UpdateImageURLs() (int64, error) {
	query := `
        UPDATE categories
        SET image = REPLACE(
            image,
            'https://storage.googleapis.com/m_tickets',
            'https://bucket.emalify.com'
        )
        WHERE image LIKE 'https://storage.googleapis.com/m_tickets%';`

	res, err := DB.Exec(query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
