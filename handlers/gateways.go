package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/services/payment_gateways"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ConfigurePaymentGatewayHandler creates or updates tenant payment gateway credentials (Paystack, Flutterwave, Stripe)
// @Summary      Configure Tenant Payment Gateway
// @Description  Save public key, secret key, and settings for Paystack, Flutterwave, or Stripe per tenant
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        config  body      dtos.TenantPaymentGatewayConfig  true  "Gateway configuration"
// @Success      200     {object}  map[string]interface{}
// @Router       /api/admin/payments/gateways [post]
func ConfigurePaymentGatewayHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	tenantID := fmt.Sprintf("%d", middleware.TenantIDFromContext(c.Request.Context()))
	if tenantID == "" || tenantID == "0" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Tenant ID required",
				Code:        http.StatusBadRequest,
			},
			Message:   "Tenant ID required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	var req dtos.TenantPaymentGatewayConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Invalid configuration payload: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	req.TenantID = tenantID
	if err := models.SaveTenantGatewayConfig(models.DB, &req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to save gateway configuration: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment gateway configured successfully",
			Code:        http.StatusOK,
		},
		Payload:   req,
		Message:   "Payment gateway configuration saved",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// GetPaymentGatewayConfigsHandler retrieves all configured payment gateways for a tenant
// @Summary      List Tenant Payment Gateways
// @Description  Get configurations for all payment gateways available to the current tenant
// @Tags         Payments
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/payments/gateways [get]
func GetPaymentGatewayConfigsHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	tenantID := fmt.Sprintf("%d", middleware.TenantIDFromContext(c.Request.Context()))
	configs, err := models.GetAllTenantGatewayConfigs(models.DB, tenantID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to fetch gateway configs: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Retrieved tenant gateway configs",
			Code:        http.StatusOK,
		},
		Payload:   configs,
		Message:   "Retrieved payment gateway configurations",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// InitializeCardPaymentHandler initiates an online payment using Paystack, Flutterwave, or Stripe
// @Summary      Initialize Card/Online Payment
// @Description  Create authorization checkout URL for Paystack, Flutterwave, or Stripe
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        initReq  body      dtos.PaymentInitializeRequest  true  "Payment initialization details"
// @Success      200      {object}  dtos.PaymentInitializeResponse
// @Router       /api/payments/card/initialize [post]
func InitializeCardPaymentHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	var req dtos.PaymentInitializeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Invalid payment payload: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	tenantID := fmt.Sprintf("%d", middleware.TenantIDFromContext(c.Request.Context()))
	service, err := payment_gateways.GetGatewayService(models.DB, tenantID, req.GatewayName)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to initialize gateway service: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	resp, err := service.InitializeTransaction(req)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Payment initialization failed: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment checkout initialized successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Payment initialized successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

// VerifyCardPaymentHandler verifies a transaction status with Paystack, Flutterwave, or Stripe
// @Summary      Verify Card/Online Payment
// @Description  Verify payment completion status using reference code
// @Tags         Payments
// @Produce      json
// @Param        gateway    query     string  true  "Gateway Name ('paystack', 'flutterwave', 'stripe')"
// @Param        reference  query     string  true  "Transaction Reference or Session ID"
// @Success      200        {object}  dtos.PaymentVerifyResponse
// @Router       /api/payments/card/verify [get]
func VerifyCardPaymentHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)

	gatewayName := c.Query("gateway")
	reference := c.Query("reference")

	if gatewayName == "" || reference == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Gateway name and reference are required",
				Code:        http.StatusBadRequest,
			},
			Message:   "Gateway name and reference are required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	tenantID := fmt.Sprintf("%d", middleware.TenantIDFromContext(c.Request.Context()))
	service, err := payment_gateways.GetGatewayService(models.DB, tenantID, gatewayName)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to initialize gateway service: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	verifyResp, err := service.VerifyTransaction(reference)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Payment verification failed: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment verification complete",
			Code:        http.StatusOK,
		},
		Payload:   verifyResp,
		Message:   "Payment verification status retrieved",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
