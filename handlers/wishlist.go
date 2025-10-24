package handlers

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"

	"github.com/gorilla/mux"
)

var noUser = "User is not validated"

//This code provides code for Wishlist and Gifts endpoints. All endpoints require user authentication for creation and modifications

// AddToWishList godoc
// @Summary Add product to wishlist
// @Description Adds a product to the user's wishlist
// @Tags Wishlist
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param wishlist body dtos.WishlistItem true "Wishlist item to add"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/wishlist/product [post]
func AddToWishList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	item, ok := DecodeRequestBody[dtos.CreateWishlistItem](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(item, w, r, requestSummary, start, "Products") {
		return
	}
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//Get wishlist, if user has no wishlist, create one and  add items to it
	wishlistID, err := models.GetOrCreateWishlist(user.ID, "My Wishlist")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get or create wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	err = models.CreateWishListItem(wishlistID, item.ProductID, user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create wishlist item for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	myWishlist, err := models.GetMyWishlistItems(user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist items for user with ID " + user.ID,
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
			Module:      "Products",
			Description: "Product added successfully to the wishlist for user with ID " + user.ID,
		},
		Payload:   myWishlist,
		Message:   "Product added successfully to the wishlist",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// RemoveFromWishList godoc
// @Summary Remove product from wishlist
// @Description Removes a product from the user's wishlist
// @Tags Wishlist
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param wishlist body dtos.WishlistItem true "Wishlist item to remove"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/wishlist/prodcut [delete]
func RemoveFromWishList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusUnauthorized,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	productID := mux.Vars(r)["product_id"]

	wishlistID, err := models.GetWishlistByUserID(user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	err = models.RemoveWishlistItem(wishlistID, productID, user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to remove wishlist item for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	myWishlist, err := models.GetMyWishlistItems(user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist items for user with ID " + user.ID,
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
			Module:      "Products",
			Description: "Product removed successfully from the wishlist for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   myWishlist,
		Message:   "Product removed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetAllUserWishList godoc
// @Summary Get all wishlists for a user
// @Description Returns all wishlists for the authenticated user
// @Tags Wishlist
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/wishlist [get]
func GetAllUserWishList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	limit := 0
	page := 1
	wishlistID := mux.Vars(r)["wishlist_id"]
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("size")
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	} else {
		limit = 10
	}
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}
	wishlists, pagination, err := models.GetAllUserWishList(user.ID, wishlistID, limit, page)
	if err != nil {
		log.Printf("no wishlist:: %s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlists for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   "WishList was not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	response := map[string]interface{}{
		"wishlists": wishlists,
	}
	if pagination != nil {
		response["pagination"] = pagination
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlists fetched successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "All wishlists",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CreateWishList godoc
// @Summary Create a new wishlist
// @Description Creates a new wishlist for the user
// @Tags Wishlist
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param wishlist body dtos.Wishlist true "Wishlist data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/wishlist [post]
func CreateWishList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	body, ok := DecodeRequestBody[dtos.CreateWishlist](r, w, requestSummary, start)
	if !ok {
		return
	}
	newList, err := models.CreateWishList(*body, user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to create wishlist",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlist created successfully for user with ID " + user.ID,
			Code:        http.StatusCreated,
		},
		Payload:   newList,
		Message:   "Wishlist created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CreateShareableWishlistLink godoc
// @Summary Generate a shareable wishlist link
// @Description Returns a public link to share a wishlist by ID
// @Tags Wishlist
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/wishlist/share [post]
func SendWishlistToShare(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Check if wishlist exists and is public
	wishlists, err := models.GetMyWishlistItems(user.ID)

	if err != nil || len(wishlists.Products) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist not found for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   "Wishlist not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	if !wishlists.IsPublic {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist is private for user with ID " + user.ID,
				Code:        http.StatusForbidden,
			},
			Message:   "This wishlist is private",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	baseURL := os.Getenv("BASE_URL")
	encodedID := base64.URLEncoding.EncodeToString([]byte(wishlists.WishlistID))
	shareLink := fmt.Sprintf("%swishlist/share/%s", baseURL, encodedID)

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Share link generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   shareLink,
		Message:   "Share link",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetSharedWishlist godoc
// @Summary Get shared wishlist
// @Description Returns a public wishlist and its products by ID
// @Tags Wishlist
// @Produce json
// @Param wishlist_id path string true "Wishlist ID"
// @Success 200 {object} dtos.Wishlist
// @Failure 404 {object} map[string]string
// @Router /api/wishlist/share/{wishlist_id} [get]
func ReceiceWishlistShared(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	vars := mux.Vars(r)
	encodedID := vars["wishlist_id"]

	decodedBytes, err := base64.URLEncoding.DecodeString(encodedID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Invalid wishlist link",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid wishlist link",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	wishlistID := string(decodedBytes)

	wishlists, err := models.GetWishlistByID(wishlistID)

	if err != nil || len(wishlists) == 0 {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist not found or is empty with ID " + wishlistID,
				Code:        http.StatusNotFound,
			},
			Message:   "Wishlist not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	wishlist := wishlists[0]
	if !wishlist.IsPublic {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist with ID " + wishlistID + " is private",
				Code:        http.StatusForbidden,
			},
			Message:   "This wishlist is private",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Shared wishlist fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   wishlist.Products,
		Message:   "Shared wishlist",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete godoc
// @Summary Delete  new wishlist
// @Description Deletes a wishlist for the user
// @Tags Wishlist
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/wishlist/{wishlist_id} [DELETE]
func DeleteWishList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	wishlistID := mux.Vars(r)["wishlist_id"]
	err := models.DeleteWishList(wishlistID, user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete wishlist with ID " + wishlistID + " for user with ID " + user.ID,
				Code:        http.StatusNotFound,
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
			Module:      "Products",
			Description: "Wishlist with ID " + wishlistID + " deleted successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Wishlist deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetMyWishList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	myWishlist, err := models.GetMyWishlistItems(user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to retrieve wishlist items for user with ID " + user.ID,
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
			Module:      "Products",
			Description: "Wishlist items retrieved successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   myWishlist,
		Message:   "My wishlists",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
