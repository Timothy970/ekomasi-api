package dtos

type AllWishlist struct {
	WishlistID string    `json:"wishlist_id"`
	Name       string    `json:"name"`
	IsPublic   bool      `json:"is_public"`
	Products   []Product `json:"products"`
}

type Wishlist struct {
	WishlistID string `json:"wishlist_id"`
	Name       string `json:"name"`
	IsPublic   bool   `json:"is_public"`
}
type CreateWishlist struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
}

type WishlistItem struct {
	ProductID string `json:"product_id" validate:"required"`
}
type CreateWishlistItem struct {
	ProductID string `json:"product_id" validate:"required"`
}
type ShareWishlistRequest struct {
	WishlistID string `json:"wishlist_id"`
}
