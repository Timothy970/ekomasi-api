package payment_gateways

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"fmt"
	"os"
	"strings"
)

type PaymentGateway interface {
	InitializeTransaction(req dtos.PaymentInitializeRequest) (*dtos.PaymentInitializeResponse, error)
	VerifyTransaction(reference string) (*dtos.PaymentVerifyResponse, error)
}

func GetGatewayService(db models.DBExecutor, tenantID, gatewayName string) (PaymentGateway, error) {
	gatewayName = strings.ToLower(gatewayName)

	// Try fetching tenant configuration from DB
	cfg, err := models.GetTenantGatewayConfig(db, tenantID, gatewayName)
	var secretKey, publicKey string

	if err == nil && cfg.IsEnabled && cfg.SecretKey != "" {
		secretKey = cfg.SecretKey
		publicKey = cfg.PublicKey
	} else {
		// Fallback to environment variables
		switch gatewayName {
		case "paystack":
			secretKey = os.Getenv("PAYSTACK_SECRET_KEY")
			publicKey = os.Getenv("PAYSTACK_PUBLIC_KEY")
		case "flutterwave":
			secretKey = os.Getenv("FLUTTERWAVE_SECRET_KEY")
			publicKey = os.Getenv("FLUTTERWAVE_PUBLIC_KEY")
		case "stripe":
			secretKey = os.Getenv("STRIPE_SECRET_KEY")
			publicKey = os.Getenv("STRIPE_PUBLIC_KEY")
		}
	}

	if secretKey == "" {
		return nil, fmt.Errorf("payment gateway '%s' is not enabled or credentials are missing for tenant %s", gatewayName, tenantID)
	}

	switch gatewayName {
	case "paystack":
		return NewPaystackService(secretKey, publicKey), nil
	case "flutterwave":
		return NewFlutterwaveService(secretKey, publicKey), nil
	case "stripe":
		return NewStripeService(secretKey, publicKey), nil
	default:
		return nil, fmt.Errorf("unsupported payment gateway '%s'", gatewayName)
	}
}
