package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

var fetchblog = "blog_id = ?"
var noblog = "blog not found"
var whereID = "id = ?"

func GetFooterData() ([]dtos.Footer, error) {
	rows, err := DB.Query("SELECT copyright_text, company_address, contact_email, phone_number FROM contact_info LIMIT 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []dtos.Footer
	for rows.Next() {
		var f dtos.Footer
		if err := rows.Scan(&f.CopyrightText, &f.CompanyAddress, &f.ContactEmail, &f.PhoneNumber); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, nil
}

func GetSocialsData() ([]dtos.SocialLink, error) {
	rows, err := DB.Query("SELECT platform, url, icon_class FROM social_links ORDER BY display_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []dtos.SocialLink
	for rows.Next() {
		var l dtos.SocialLink
		if err := rows.Scan(&l.Platform, &l.URL, &l.IconClass); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

func GetMenuData() ([]dtos.MenuLink, error) {
	rows, err := DB.Query("SELECT  title, path FROM static_pages ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []dtos.MenuLink
	for rows.Next() {
		var l dtos.MenuLink
		if err := rows.Scan(&l.Title, &l.HREF); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

func GetBannersData(value string) ([]dtos.Banner, error) {
	rows, err := DB.Query(`
		SELECT id, image_url, text, heading, button_text, button_url, display_order, is_active, type
		FROM banners WHERE is_active = true AND type = ? ORDER BY display_order ASC`, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var banners []dtos.Banner
	for rows.Next() {
		var b dtos.Banner
		err := rows.Scan(&b.ID, &b.ImageURL, &b.Text, &b.Heading, &b.ButtonText, &b.ButtonURL, &b.DisplayOrder, &b.IsActive, &b.Type)
		if err != nil {
			return nil, err
		}
		banners = append(banners, b)
	}
	return banners, nil
}
func GetPromotions() ([]dtos.Promotion, error) {
	now := time.Now()
	promotionQuery := `
	SELECT 
		p.promotion_id, 
		p.name, 
		p.start_date, 
		p.end_date, 
		p.is_active, 
		pt.name AS promotion_type, 
		pt.description AS promotion_description,
		pt.value AS amount
	FROM promotions p
	LEFT JOIN promotion_types pt ON pt.id = p.promotion_type_id
	WHERE p.is_active = TRUE 
	  AND p.start_date <= ? 
	  AND p.end_date >= ?
`

	rows, err := DB.Query(promotionQuery, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []dtos.Promotion

	for rows.Next() {
		var promo dtos.Promotion
		if err := rows.Scan(&promo.ID, &promo.Name, &promo.StartDate, &promo.EndDate, &promo.IsActive, &promo.PromotionType, &promo.PromotionDescription, &promo.Amount); err != nil {
			return nil, err
		}
		log.Printf("Got promotion ::::::::%v", promo)
		products, err := getPromotionProductGroups(promo.ID)
		if err != nil {
			return nil, err
		}

		promo.PromotionProducts = products
		promotions = append(promotions, promo)
	}
	log.Printf("Got promotion ::::::::%v", promotions)

	return promotions, nil
}

func getPromotionProductGroups(promotionID string) ([]dtos.PromotionProductGroup, error) {
	query := `
		SELECT promotion_product_id, promotion_id, product_id 
		FROM promotion_products 
		WHERE promotion_id = ?
	`

	rows, err := DB.Query(query, promotionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []dtos.PromotionProductGroup

	for rows.Next() {
		var pp dtos.PromotionProductGroup
		if err := rows.Scan(&pp.ID, &pp.PromotionID, &pp.ProductID); err != nil {
			return nil, err
		}

		product, category, err := getProductWithCategory(pp.ProductID)
		if err != nil {
			return nil, err
		}

		// Add the product under its category
		catMap := make(map[string]*dtos.CategoryGroup)
		catID := category.CategoryID
		if _, exists := catMap[catID]; !exists {
			catMap[catID] = &dtos.CategoryGroup{
				CategoryID:       category.CategoryID,
				Name:             category.Name,
				ParentCategoryID: category.ParentCategoryID,
				Description:      category.Description,
				Products:         []dtos.Product{},
			}
		}
		catMap[catID].Products = append(catMap[catID].Products, product)

		// Convert map to slice
		for _, cat := range catMap {
			pp.Categories = append(pp.Categories, *cat)
		}

		groups = append(groups, pp)
	}

	return groups, nil
}

func getProductWithCategory(productID string) (dtos.Product, dtos.CategoryGroup, error) {
	query := `
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id, 
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at,
			c.category_id, c.name, c.parent_category_id, c.description
		FROM products p
		JOIN categories c ON p.category_id = c.category_id
		WHERE p.product_id = ?
	`

	var product dtos.Product
	var category dtos.CategoryGroup

	row := DB.QueryRow(query, productID)
	err := row.Scan(
		&product.ID, &product.Name, &product.Description, &product.SKU, &product.Price,
		&product.CategoryID, &product.StockQuantity, &product.SearchVector,
		&product.CreatedAt, &product.LastUpdated,
		&category.CategoryID, &category.Name, &category.ParentCategoryID, &category.Description,
	)
	if err != nil {
		return dtos.Product{}, dtos.CategoryGroup{}, err
	}

	// Fetch product images
	imageRows, err := DB.Query(`
		SELECT image_id, url, is_primary 
		FROM product_images 
		WHERE product_id = ?`, productID,
	)
	if err != nil {
		return dtos.Product{}, dtos.CategoryGroup{}, err
	}
	defer imageRows.Close()

	var images []dtos.Image
	for imageRows.Next() {
		var img dtos.Image
		if err := imageRows.Scan(&img.ImageID, &img.URL, &img.IsPrimary); err != nil {
			return dtos.Product{}, dtos.CategoryGroup{}, err
		}
		images = append(images, img)
	}
	product.Images = images

	return product, category, nil
}

// func groupByCategory(details []struct {
// 	dtos.PromotionProduct
// 	Product  dtos.Product
// 	Category dtos.Category
// }) []dtos.CategoryGroup {
// 	categoryMap := make(map[string]*dtos.CategoryGroup)

// 	for _, d := range details {
// 		catID := d.Category.ID
// 		if _, exists := categoryMap[catID]; !exists {
// 			categoryMap[catID] = &dtos.CategoryGroup{
// 				CategoryID:       d.Category.ID,
// 				Name:             d.Category.Name,
// 				ParentCategoryID: d.Category.ParentCategoryID,
// 				Description:      d.Category.Description,
// 				Products:         []dtos.Product{},
// 			}
// 		}
// 		categoryMap[catID].Products = append(categoryMap[catID].Products, d.Product)
// 	}

// 	var categories []dtos.CategoryGroup
// 	for _, c := range categoryMap {
// 		categories = append(categories, *c)
// 	}

// 	return categories
// }

func GetCategoriesWithProducts() ([]dtos.CategoryWithProducts, error) {
	rows, err := DB.Query(`
		SELECT 
			c.category_id, c.name, c.parent_category_id, c.description,
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
		FROM categories c
		LEFT JOIN products p ON c.category_id = p.category_id
		ORDER BY c.category_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categoryMap := make(map[string]*dtos.CategoryWithProducts)

	for rows.Next() {
		catID, catName, catDesc, parentCatID, product, err := scanCategoryProductRow(rows)
		if err != nil {
			return nil, err
		}

		// Ensure category is in map
		if _, exists := categoryMap[catID]; !exists {
			categoryMap[catID] = &dtos.CategoryWithProducts{
				CategoryID:       catID,
				Name:             catName,
				ParentCategoryID: parentCatID,
				Description:      catDesc,
				Products:         []dtos.Product{},
			}
		}

		// If product is not null, load images and append
		if product.ID != "" {
			images, err := fetchProductImages(product.ID)
			if err != nil {
				return nil, err
			}
			product.Images = images
			categoryMap[catID].Products = append(categoryMap[catID].Products, product)
		}
	}

	// Convert map to slice
	var result []dtos.CategoryWithProducts
	for _, cat := range categoryMap {
		result = append(result, *cat)
	}

	return result, nil
}
func scanCategoryProductRow(rows *sql.Rows) (string, string, string, *string, dtos.Product, error) {
	var (
		catID, catName, catDesc string
		parentCatID             *string
		p                       dtos.Product
	)
	err := rows.Scan(
		&catID, &catName, &parentCatID, &catDesc,
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
	)
	return catID, catName, catDesc, parentCatID, p, err
}

func fetchProductImages(productID string) ([]dtos.Image, error) {
	rows, err := DB.Query(`
		SELECT image_id, url, is_primary, type
		FROM product_images 
		WHERE product_id = ?`, productID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []dtos.Image
	for rows.Next() {
		var img dtos.Image
		if err := rows.Scan(&img.ImageID, &img.URL, &img.IsPrimary, &img.Type); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}
func fetchProductFeatures(productID string) ([]dtos.ProductFeature, error) {
	query := `
		SELECT feature_id, product_id, header, image, description, image_position
		FROM product_features
		WHERE product_id = ?
		ORDER BY created_at ASC`
	rows, err := DB.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []dtos.ProductFeature
	for rows.Next() {
		var feature dtos.ProductFeature
		if err := rows.Scan(&feature.ID, &feature.ProductID, &feature.Header, &feature.Image, &feature.Description, &feature.ImagePosition); err != nil {
			return nil, err
		}
		features = append(features, feature)
	}
	return features, nil
}

func InsertBannerDetails(url string, req dtos.BannerInfo) error {
	query := `INSERT INTO banners (image_url, text, heading, button_text, button_url, display_order, is_active, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := DB.Exec(query, url, req.Text, req.Heading, req.ButtonText, req.ButtonURL, req.DisplayOrder, req.IsActive, req.Type)
	return err
}

func UpdateBannerDetails(req dtos.UpdateBannerInfo, bannerID string) error {
	exists, err := RecordExists("banners", whereID, bannerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("banner not found")
	}
	query := "UPDATE banners SET"
	args := []interface{}{}
	updates := []string{}

	if req.Text != "" {
		updates = append(updates, "text = ?")
		args = append(args, req.Text)
	}
	if req.Heading != "" {
		updates = append(updates, "heading = ?")
		args = append(args, req.Heading)
	}
	if req.ButtonText != "" {
		updates = append(updates, "button_text = ?")
		args = append(args, req.ButtonText)
	}
	if req.ButtonURL != "" {
		updates = append(updates, "button_url = ?")
		args = append(args, req.ButtonURL)
	}
	if req.DisplayOrder != 0 {
		updates = append(updates, "display_order = ?")
		args = append(args, req.DisplayOrder)
	}
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, req.IsActive)
	}
	if len(updates) == 0 {
		return fmt.Errorf("request cannot be empty") // Nothing to update
	}

	query += " " + strings.Join(updates, ", ") + " WHERE id = ?"
	args = append(args, bannerID)

	if _, err := DB.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update review: %v", err)
	}

	return nil
}

// fetch promotion types
func GetPromotionsTypes() ([]dtos.PromotionType, error) {
	rows, err := DB.Query(`
		SELECT id, name, description, value 
		FROM promotion_types 
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []dtos.PromotionType
	for rows.Next() {
		var typesingle dtos.PromotionType
		if err := rows.Scan(&typesingle.ID, &typesingle.Name, &typesingle.Description, &typesingle.Value); err != nil {
			return nil, err
		}
		types = append(types, typesingle)
	}
	return types, nil
}

// insert new promotion in the database
func CreateNewPromotion(req dtos.NewPromotion) (string, error) {
	// Generate cart ID
	promotionID, _ := shortid.Generate()
	// Insert the item
	_, err := DB.Exec(`
        INSERT INTO promotions(promotion_id, name, promotion_type_id, start_date, end_date)
        VALUES (?, ?, ?, ?, ?)
    `, promotionID, req.Name, req.PromotionIdType, req.StartDate, req.EndDate)
	if err != nil {
		return "", fmt.Errorf("failed to promotion: %w", err)
	}

	return promotionID, nil
}

// Delete a promotion
func DeletePromotion(promotionID string) error {
	_, err := DB.Exec(`
		DELETE FROM promotions WHERE promotion_id = ?
	`, promotionID)
	return err
}
func EditPromotion(req dtos.EditPromotion) error {
	query := "UPDATE promotions SET"
	args := []interface{}{}
	updates := []string{}

	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
	}
	if req.PromotionIdType != 0 {
		updates = append(updates, "promotion_type_id = ?")
		args = append(args, req.PromotionIdType)
	}
	if !req.StartDate.IsZero() {
		updates = append(updates, "start_date = ?")
		args = append(args, req.StartDate)
	}
	if !req.EndDate.IsZero() {
		updates = append(updates, "end_date = ?")
		args = append(args, req.StartDate)
	}
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, req.IsActive)
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	query += " " + strings.Join(updates, ", ") + " WHERE promotion_id = ?"
	args = append(args, req.PromotionID)

	if _, err := DB.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update promotion: %v", err)
	}

	return nil
}

func AttachProductsToPromotion(req dtos.AttachProductToPromotion) error {
	if len(req.ProductIDs) == 0 {
		return fmt.Errorf("no product IDs provided")
	}
	checkQuery := `SELECT COUNT(1) FROM promotion_products WHERE promotion_id = ? AND product_id = ?`
	insertQuery := `INSERT INTO promotion_products (promotion_product_id, promotion_id, product_id) VALUES (?, ?, ?)`

	for _, productID := range req.ProductIDs {
		promotionProductID, _ := shortid.Generate()
		var count int
		err := DB.QueryRow(checkQuery, req.PromotionID, productID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check existence for product %s: %v", productID, err)
		}

		if count > 0 {
			continue // skip if already exists
		}

		if _, err := DB.Exec(insertQuery, promotionProductID, req.PromotionID, productID); err != nil {
			return fmt.Errorf("failed to insert product %s: %v", productID, err)
		}
	}
	return nil
}

func CheckPromotionExists(promotionID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM promotions WHERE promotion_id = ?)`
	err := DB.QueryRow(query, promotionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check promotion existence: %v", err)
	}
	return exists, nil
}
func RemoveProductFromPromotion(req dtos.AttachProductToPromotion) error {
	query := `DELETE FROM promotion_products WHERE promotion_id = ? AND product_id = ?`

	var anyDeleted bool

	for _, productID := range req.ProductIDs {
		result, err := DB.Exec(query, req.PromotionID, productID)
		if err != nil {
			return fmt.Errorf("failed to remove product %s from promotion %s: %v", productID, req.PromotionID, err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			anyDeleted = true
		}
	}

	if !anyDeleted {
		return fmt.Errorf("no matching products found to remove for promotion %s", req.PromotionID)
	}

	return nil
}

func CreateBlog(blog dtos.BlogRequest, authorID string) error {
	blogID, _ := shortid.Generate()
	var publishedAt *string
	isPublished := true
	if blog.Status == nil {
		blog.Status = new(string)
		*blog.Status = "published"
	}
	if strings.ToLower(*blog.Status) == "draft" {
		isPublished = false
		publishedAt = nil // ← represents NULL in DB
	} else {
		now := time.Now().Format("2006-01-02 15:04:05")
		publishedAt = &now

	}
	// marshal content to json
	contentData, err := json.Marshal(blog.Sections)
	if err != nil {
		fmt.Println("Error converting to JSON:", err)
		return err
	}

	authorData, err := json.Marshal(blog.Author)
	if err != nil {
		fmt.Println("Error converting author to JSON:", err)
		return err
	}
	tagData, err := json.Marshal(blog.Tags)
	if err != nil {
		fmt.Println("Error converting tags to JSON:", err)
		return err
	}
	query := `
		INSERT INTO blogs (blog_id, title, content, author_id, published_at, is_published, author, tags, description, read_time, status, banner_image_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = DB.Exec(query, blogID, blog.Title, contentData, authorID, publishedAt, isPublished, authorData, tagData, blog.Description, blog.ReadTimeMinutes, blog.Status, blog.BannerImageUrl)
	return err
}
func GetBlogByID(blogID string) (*dtos.BlogRequest, error) {
	exists, err := RecordExists("blogs", fetchblog, blogID)
	if err != nil {
		return nil, fmt.Errorf("failed to check blog existence: %w", err)
	}
	if !exists {
		return nil, errors.New(noblog)
	}

	query := `
		SELECT blog_id, title, content, author_id, published_at, is_published, 
		       author, tags, description, read_time, status, created_at, updated_at, banner_image_url
		FROM blogs
		WHERE blog_id = ?
	`

	var (
		blog        dtos.BlogRequest
		contentJSON sql.NullString
		authorJSON  sql.NullString
		tagsJSON    sql.NullString
		publishedAt sql.NullTime
		readTime    sql.NullInt64
	)

	row := DB.QueryRow(query, blogID)
	if err := row.Scan(
		&blog.BlogID, &blog.Title, &contentJSON, &blog.AuthorID,
		&publishedAt, &blog.IsPublished, &authorJSON, &tagsJSON,
		&blog.Description, &readTime, &blog.Status,
		&blog.CreatedAt, &blog.UpdatedAt, &blog.BannerImageUrl,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New(noblog)
		}
		return nil, fmt.Errorf("failed to scan blog: %w", err)
	}

	unmarshalJSONField := func(data sql.NullString, target interface{}, field string) error {
		if data.Valid {
			if err := json.Unmarshal([]byte(data.String), target); err != nil {
				return fmt.Errorf("invalid %s JSON: %w", field, err)
			}
		}
		return nil
	}

	if err := unmarshalJSONField(contentJSON, &blog.Sections, "content"); err != nil {
		return nil, err
	}
	if err := unmarshalJSONField(authorJSON, &blog.Author, "author"); err != nil {
		return nil, err
	}
	if err := unmarshalJSONField(tagsJSON, &blog.Tags, "tags"); err != nil {
		return nil, err
	}

	if publishedAt.Valid {
		blog.PublishedAt = publishedAt.Time
	}
	if readTime.Valid {
		blog.ReadTimeMinutes = int(readTime.Int64)
	}

	return &blog, nil
}

func UpdateBlog(blog dtos.BlogRequest, blogID string) error {
	exists, err := RecordExists("blogs", fetchblog, blogID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noblog)
	}
	var publishedAt *string
	isPublished := true
	if blog.Status == nil {
		blog.Status = new(string)
		*blog.Status = "published"
	}
	if strings.ToLower(*blog.Status) == "draft" {
		isPublished = false
		publishedAt = nil // ← represents NULL in DB
	} else {
		now := time.Now().Format("2006-01-02 15:04:05")
		publishedAt = &now

	}
	// marshal content to json
	contentData, err := json.Marshal(blog.Sections)
	if err != nil {
		fmt.Println("Error converting to JSON:", err)
		return err
	}

	authorData, err := json.Marshal(blog.Author)
	if err != nil {
		fmt.Println("Error converting author to JSON:", err)
		return err
	}
	tagData, err := json.Marshal(blog.Tags)
	if err != nil {
		fmt.Println("Error converting tags to JSON:", err)
		return err
	}
	query := `
		UPDATE blogs SET title = ?, content = ?, published_at = ?, is_published = ?, author = ?, tags = ?, description = ?, read_time = ?, status = ?, banner_image_url = ?
		WHERE blog_id = ?`

	_, err = DB.Exec(query, blog.Title, contentData, publishedAt, isPublished, authorData, tagData, blog.Description, blog.ReadTimeMinutes, blog.Status, blog.BannerImageUrl, blogID)
	return err
}
func DeleteBlog(blogID string) error {
	exists, err := RecordExists("blogs", fetchblog, blogID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noblog)
	}
	query := `DELETE FROM blogs WHERE blog_id = ?`
	_, err = DB.Exec(query, blogID)
	return err
}
func ListBlogs(page, limit int, status string) ([]dtos.BlogRequest, *dtos.PaginationMeta, error) {
	offset := (page - 1) * limit

	totalItems, err := countBlogs(status)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count blogs: %w", err)
	}

	rows, err := fetchBlogs(status, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query blogs: %w", err)
	}
	defer rows.Close()

	blogs, err := scanBlogs(rows)
	if err != nil {
		return nil, nil, err
	}

	meta := buildPagination(limit, offset, totalItems)
	return blogs, meta, nil
}
func countBlogs(status string) (int, error) {
	query := "SELECT COUNT(*) FROM blogs"
	var args []interface{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	var total int
	if err := DB.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
func fetchBlogs(status string, limit, offset int) (*sql.Rows, error) {
	query := `
		SELECT blog_id, title, content, author_id, published_at, is_published, 
		       author, tags, description, read_time, status, created_at, updated_at, banner_image_url
		FROM blogs
	`
	var args []interface{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY published_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	return DB.Query(query, args...)
}
func scanBlogs(rows *sql.Rows) ([]dtos.BlogRequest, error) {
	var blogs []dtos.BlogRequest

	for rows.Next() {
		var (
			blog        dtos.BlogRequest
			contentJSON sql.NullString
			authorJSON  sql.NullString
			tagsJSON    sql.NullString
			publishedAt sql.NullTime
			readTime    sql.NullInt64
		)

		if err := rows.Scan(
			&blog.BlogID, &blog.Title, &contentJSON, &blog.AuthorID,
			&publishedAt, &blog.IsPublished, &authorJSON, &tagsJSON,
			&blog.Description, &readTime, &blog.Status,
			&blog.CreatedAt, &blog.UpdatedAt, &blog.BannerImageUrl,
		); err != nil {
			return nil, fmt.Errorf("failed to scan blog: %w", err)
		}

		if err := parseBlogFields(&blog, contentJSON, authorJSON, tagsJSON, publishedAt, readTime); err != nil {
			return nil, err
		}

		blogs = append(blogs, blog)
	}
	return blogs, nil
}
func parseBlogFields(
	blog *dtos.BlogRequest,
	contentJSON, authorJSON, tagsJSON sql.NullString,
	publishedAt sql.NullTime,
	readTime sql.NullInt64,
) error {
	unmarshal := func(data sql.NullString, target interface{}, field string) error {
		if data.Valid {
			if err := json.Unmarshal([]byte(data.String), target); err != nil {
				return fmt.Errorf("invalid %s JSON: %w", field, err)
			}
		}
		return nil
	}

	if err := unmarshal(contentJSON, &blog.Sections, "content"); err != nil {
		return err
	}
	if err := unmarshal(authorJSON, &blog.Author, "author"); err != nil {
		return err
	}
	if err := unmarshal(tagsJSON, &blog.Tags, "tags"); err != nil {
		return err
	}

	if publishedAt.Valid {
		blog.PublishedAt = publishedAt.Time
	}
	if readTime.Valid {
		blog.ReadTimeMinutes = int(readTime.Int64)
	}
	return nil
}

func DeleteBanner(bannerID string) error {
	exists, err := RecordExists("banners", whereID, bannerID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("banner not found")
	}
	query := `DELETE FROM banners WHERE id = ?`
	_, err = DB.Exec(query, bannerID)
	return err
}

func CreateMenuLink(req dtos.MenuLinkRequest) error {
	exists, err := RecordExists("menu_links", "title = ?", req.Title)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("menu title already exists")
	}

	// If ParentID is nil, we insert first then set it to self
	if req.ParentID == nil {
		// Insert without parent_id
		query := `INSERT INTO menu_links (title, url, display_order) VALUES (?, ?, ?)`
		_, err := DB.Exec(query, req.Title, req.URL, req.DisplayOrder)
		if err != nil {
			return err
		}

		// // Get inserted ID
		// id, err := result.LastInsertId()
		// if err != nil {
		// 	return err
		// }

		// // Update parent_id to self
		// _, err = DB.Exec(`UPDATE menu_links SET parent_id = ? WHERE id = ?`, id, id)
		// if err != nil {
		// 	return err
		// }

	} else {
		// Insert with provided parent_id
		query := `INSERT INTO menu_links (title, url, display_order, parent_id) VALUES (?, ?, ?, ?)`
		_, err = DB.Exec(query, req.Title, req.URL, req.DisplayOrder, req.ParentID)
		if err != nil {
			return err
		}
	}

	return nil
}

func UpdateMenuLink(menu dtos.MenuLinkRequest, menuLinkID int) error {
	exists, err := RecordExists("menu_links", whereID, menuLinkID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("menu link not found")
	}
	query := `UPDATE menu_links SET title=?, url=?, display_order=? WHERE id=?`
	_, err = DB.Exec(query, menu.Title, menu.URL, menu.DisplayOrder, menuLinkID)
	if err != nil {
		return err
	}
	return nil
}
func DeleteMenuLink(menuLinkID int) error {
	exists, err := RecordExists("menu_links", whereID, menuLinkID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("menu link not found")
	}
	query := `DELETE FROM menu_links WHERE id=?`
	_, err = DB.Exec(query, menuLinkID)
	return err
}

func CreateSocialLink(link *dtos.SocialLinkRequest) error {
	exists, err := RecordExists("social_links", "platform = ?", link.Platform)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("platform already exists")
	}
	query := `INSERT INTO social_links (platform, url, icon_class, display_order) VALUES (?, ?, ?, ?)`
	_, err = DB.Exec(query, link.Platform, link.URL, link.IconClass, link.DisplayOrder)
	if err != nil {
		return err
	}

	return nil
}
func UpdateSocialLink(socialID int, link dtos.SocialLinkRequest) error {
	exists, err := RecordExists("social_links", whereID, socialID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("social not found")
	}
	query := `UPDATE social_links SET platform = ?, url = ?, icon_class = ?, display_order = ? WHERE id = ?`
	_, err = DB.Exec(query, link.Platform, link.URL, link.IconClass, link.DisplayOrder, socialID)
	return err
}

func DeleteSocialLink(id int) error {
	exists, err := RecordExists("social_links", whereID, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("social not found")
	}
	query := `DELETE FROM social_links WHERE id = ?`
	_, err = DB.Exec(query, id)
	return err
}

func AddFeaturedProduct(productID string) error {
	exists, err := RecordExists("products", "product_id =? ", productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not found")
	}
	query := `INSERT INTO featured_products (product_id) VALUES (?)`
	_, err = DB.Exec(query, productID)
	return err
}

func RemoveFeaturedProduct(productID string) error {
	exists, err := RecordExists("products", "product_id =? ", productID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("product not found")
	}
	query := `DELETE FROM featured_products WHERE product_id = ?`
	_, err = DB.Exec(query, productID)
	return err
}

func GetFeaturedProducts() ([]dtos.Product, error) {
	rows, err := DB.Query(`
		SELECT 
			p.product_id, p.name, p.description, p.sku, p.price, p.category_id,
			p.stock_quantity, p.search_vector, p.created_at, p.last_updated_at
		FROM featured_products fp
		JOIN products p ON fp.product_id = p.product_id
		ORDER BY fp.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var featured []dtos.Product

	for rows.Next() {
		product, err := scanFeaturedProductRow(rows)
		if err != nil {
			return nil, err
		}

		// Load images for this product
		images, err := fetchProductImages(product.ID)
		if err != nil {
			return nil, err
		}
		product.Images = images

		featured = append(featured, product)
	}

	return featured, nil
}
func scanFeaturedProductRow(rows *sql.Rows) (dtos.Product, error) {
	var p dtos.Product
	err := rows.Scan(
		&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CategoryID,
		&p.StockQuantity, &p.SearchVector, &p.CreatedAt, &p.LastUpdated,
	)
	return p, err
}
