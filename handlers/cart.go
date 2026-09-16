package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

var (
	notAuthenticated     = "User not authenticated"
	failedToGetCartItems = "Failed to get cart items for Cart ID "
)

// CreateCartHandler creates a new shopping cart for the user.
//
// @Summary      Create Cart
// @Description  Create a new shopping cart for the user.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        cart  body      dtos.CreateCartRequest  true  "Create Cart Request"
// @Success      200   {object}  dtos.AddToCartResponse
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart [post]
func CreateCartHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.CreateCartRequest](c, requestSummary, start)
	if !ok {
		return
	}
	// if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start) {
	// 	return
	// }

	// Create cart in database
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	cartID, err := models.CreateCart(models.DB, *req, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to create cart",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with created cart ID
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Cart created successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"cart_id": cartID},
		Message:   "Cart created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetUserCartHandler retrieves the cart for the authenticated user.
//
// @Summary      Get Cart
// @Description  Retrieve the shopping cart for the currently authenticated user.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Success      200  {object}  dtos.AddToCartResponse
// @Failure      400  {object}  dtos.ErrorResponse
// @Failure      403  {object}  dtos.ErrorResponse
// @Failure      500  {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart [get]
func GetUserCartHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	user, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "User not authenticated to fetch cart",
				Code:        http.StatusForbidden,
			},
			Message:   notAuthenticated,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch user's cart ID
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	cartID, err := models.GetUserCart(models.DB, user.ID, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to fetch user cart with user ID" + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with cart ID
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Cart with ID " + cartID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"cart_id": cartID},
		Message:   "Cart fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AddToCartHandler adds a product to the user's cart.
//
// @Summary      Add to Cart
// @Description  Add a product item to the user's shopping cart.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        cart  body      dtos.AddToCartRequest  true  "Cart Item Details"
// @Success      200   {object}  dtos.AddToCartResponse
// @Failure      400   {object}  dtos.ErrorResponse
// @Failure      404   {object}  dtos.ErrorResponse
// @Failure      500   {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart/add [post]
func AddToCartHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	req, ok := DecodeRequestBody[dtos.AddToCartRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Cart") {
		return
	}

	// Insert item into cart
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
	if err := models.InsertCartItem(models.DB, req.CartID, req.ProductID, req.Quantity, req.VariationSKU, tenantID); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: "Failed to add item to cart with Cart ID " + req.CartID,
				Code:        404,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch updated cart items
	res, err := getCartItemsByCartID(req.CartID, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Cart",
				Description: failedToGetCartItems + req.CartID,
				Code:        404,
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
			Description: "Product with ID " + req.ProductID + " added to cart successfully with Cart ID " + req.CartID,
			Code:        http.StatusOK,
		},
		Payload:   res,
		Message:   "Product added to cart successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ViewCartHandler retrieves the contents of a specific cart.
//
// @Summary      View Cart
// @Description  Retrieve current items in the specified cart.
// @Tags         Cart
// @Produce      json
// @Param        cart_id      path      string  true   "Cart ID"
// @Param        location_id  query     int     false  "Location ID for delivery charge calculation"
// @Success      200          {object}  dtos.ViewCartResponse
// @Failure      400          {object}  dtos.ErrorResponse
// @Failure      500          {object}  dtos.ErrorResponse
// @Security     BearerAuth
// @Router       /api/cart/view/{cart_id} [get]
func ViewCartHandler(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract cart ID from path variables
	cartID := c.Param("cart_id")
	locationID := c.Query("location_id")

	// Fetch cart items
	tenantID := middleware.TenantIDFromContext(c.Request.Context())
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

	// Calculate delivery charge if location ID is provided
	if locationID != "" {
		locationIDInt, _ := strconv.Atoi(locationID)
		if locationIDInt != 0 {
			loc, err := models.GetLocationByID(models.DB, locationIDInt)
			if err != nil {
				utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Cart",
						Description: "Failed to fetch location with ID " + locationID,
						Code:        http.StatusInternalServerError,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   c.Request,
					RawBody:   requestSummary})
				return
			}
			res.DeliverCharge = loc.Charge
			res.TotalAmount += loc.Charge

		}
	}

	// Respond with cart items
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Cart",
			Description: "Cart Items fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   res,
		Message:   "Cart Items fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
