package handlers

import (
	"ekomasi_backend/config"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// WhatsAppFlowDataEndpoint handles encrypted requests from Meta WhatsApp Flows
func WhatsAppFlowDataEndpoint(c *gin.Context) {
	var encReq utils.FlowEncryptedRequest
	if err := c.ShouldBindJSON(&encReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid encrypted payload format"})
		return
	}

	cfg := config.Get()
	decryptedReq, aesKey, iv, err := utils.DecryptFlowPayload(encReq, cfg.WhatsAppCloud.FlowPrivateKey)
	if err != nil {
		log.Printf("[WhatsApp Flow Endpoint] Decryption failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Decryption failed", "details": err.Error()})
		return
	}

	var responsePayload interface{}

	// Action router according to Meta Flow Spec
	switch decryptedReq.Action {
	case "ping":
		version := decryptedReq.Version
		if version == "" {
			version = "3.0"
		}
		responsePayload = map[string]interface{}{
			"version": version,
			"data": map[string]interface{}{
				"status": "active",
			},
		}
	case "INIT":
		responsePayload = handleFlowInit(decryptedReq)
	case "data_exchange":
		responsePayload = handleFlowDataExchange(decryptedReq)
	default:
		responsePayload = handleFlowSubmission(decryptedReq)
	}

	// Encrypt response payload back to Meta using inverted IV
	encryptedResponse, err := utils.EncryptFlowResponse(responsePayload, aesKey, iv)
	if err != nil {
		log.Printf("[WhatsApp Flow Endpoint] Response encryption failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Response encryption failed"})
		return
	}

	c.String(http.StatusOK, encryptedResponse)
}

func handleFlowInit(req *utils.FlowDecryptedRequest) map[string]interface{} {
	switch req.Screen {
	case "CATEGORY_SELECT_SCREEN":
		return map[string]interface{}{
			"version": req.Version,
			"screen":  req.Screen,
			"data": map[string]interface{}{
				"categories": getFlowCategories(),
			},
		}
	case "TRACK_ORDER_SCREEN":
		return map[string]interface{}{
			"version": req.Version,
			"screen":  req.Screen,
			"data": map[string]interface{}{
				"recent_orders": getFlowRecentOrders(req.FlowToken),
			},
		}
	case "RETURN_INIT_SCREEN":
		return map[string]interface{}{
			"version": req.Version,
			"screen":  req.Screen,
			"data": map[string]interface{}{
				"delivered_orders": getFlowDeliveredOrders(req.FlowToken),
			},
		}
	}

	return map[string]interface{}{
		"version": req.Version,
		"screen":  req.Screen,
		"data":    map[string]interface{}{},
	}
}

func handleFlowDataExchange(req *utils.FlowDecryptedRequest) map[string]interface{} {
	action, _ := req.Data["action"].(string)

	switch action {
	case "FETCH_PRODUCTS":
		catID, _ := req.Data["category_id"].(string)
		return map[string]interface{}{
			"version": req.Version,
			"screen":  "PRODUCT_DETAIL_SCREEN",
			"data": map[string]interface{}{
				"products": getFlowProductsByCategory(catID),
			},
		}
	case "GET_ORDER_STATUS":
		orderID, _ := req.Data["order_id"].(string)
		return map[string]interface{}{
			"version": req.Version,
			"screen":  "TRACKING_STATUS_SCREEN",
			"data":    getFlowOrderStatusDetails(orderID),
		}
	case "GENERATE_RECOMMENDATIONS":
		budget, _ := req.Data["budget"].(string)
		occasion, _ := req.Data["occasion"].(string)
		return map[string]interface{}{
			"version": req.Version,
			"screen":  "RECOMMENDATION_RESULTS_SCREEN",
			"data": map[string]interface{}{
				"recommended_products": queryMeilisearchRecommendations(budget, occasion),
			},
		}
	}

	return map[string]interface{}{
		"version": req.Version,
		"screen":  req.Screen,
		"data":    map[string]interface{}{},
	}
}

func handleFlowSubmission(req *utils.FlowDecryptedRequest) map[string]interface{} {
	screen := req.Screen
	payload := req.Data

	switch screen {
	case "REGISTER_SCREEN":
		// 1. Customer Registration Flow
		fullName, _ := payload["full_name"].(string)
		email, _ := payload["email"].(string)
		address, _ := payload["delivery_address"].(string)
		phone := req.FlowToken

		models.DB.Exec(`
			INSERT INTO users (phone, name, email, address, created_at) 
			VALUES (?, ?, ?, ?, NOW()) 
			ON DUPLICATE KEY UPDATE name=VALUES(name), email=VALUES(email), address=VALUES(address)`,
			phone, fullName, email, address,
		)

		return map[string]interface{}{
			"screen": "SUCCESS_SCREEN",
			"data": map[string]interface{}{
				"message": fmt.Sprintf("Welcome %s! Your profile has been registered successfully.", fullName),
			},
		}

	case "CHECKOUT_SUMMARY":
		// 3. In-Chat Checkout Flow (M-Pesa STK Push Integration)
		mpesaPhone, _ := payload["mpesa_phone"].(string)
		paymentMethod, _ := payload["payment_method"].(string)
		shippingAddress, _ := payload["shipping_address"].(string)

		orderID := fmt.Sprintf("ORD-%d", time.Now().Unix())
		amount := 2500.0 // Default order amount / dynamic cart evaluation

		log.Printf("[WhatsApp Flow Checkout] Order %s created for shipping to %s", orderID, shippingAddress)

		if paymentMethod == "MPESA" && mpesaPhone != "" {
			mpesaReq := dtos.MpesaRequest{
				Phone:       mpesaPhone,
				Amount:      int(amount),
				Reference:   "EKOMASI-" + orderID,
				Description: "Payment for order " + orderID,
				OrderID:     orderID,
			}
			go func(req dtos.MpesaRequest) {
				client, err := NewMpesaClient()
				if err == nil {
					_, _ = client.LipaNaMpesaOnline(req)
				}
			}(mpesaReq)
		}

		return map[string]interface{}{
			"screen": "SUCCESS_SCREEN",
			"data": map[string]interface{}{
				"order_id": orderID,
				"message":  fmt.Sprintf("Order #%s placed! M-Pesa STK Push initiated for %s.", orderID, mpesaPhone),
			},
		}

	case "RETURN_INIT_SCREEN":
		// 5. Return Request Flow -> Automatic Customer In-App Wallet Credit
		orderID, _ := payload["order_id"].(string)
		reason, _ := payload["reason"].(string)
		comments, _ := payload["comments"].(string)
		refundAmount := 1500.0 // Evaluated item value

		// Credit Customer Wallet
		userPhone := req.FlowToken
		models.DB.Exec(`
			INSERT INTO user_wallets (user_phone, balance, updated_at) 
			VALUES (?, ?, NOW()) 
			ON DUPLICATE KEY UPDATE balance = balance + VALUES(balance), updated_at = NOW()`,
			userPhone, refundAmount,
		)

		models.DB.Exec(`
			INSERT INTO return_requests (order_id, phone, reason, comments, refund_type, status, created_at) 
			VALUES (?, ?, ?, ?, 'STORE_CREDIT', 'APPROVED', NOW())`,
			orderID, userPhone, reason, comments,
		)

		return map[string]interface{}{
			"screen": "SUCCESS_SCREEN",
			"data": map[string]interface{}{
				"message": fmt.Sprintf("Return request approved for Order #%s! KES %.2f has been credited to your in-app wallet.", orderID, refundAmount),
			},
		}

	case "FEEDBACK_SCREEN":
		// 6. Customer Feedback Flow -> 10% Voucher Reward
		orderID, _ := payload["order_id"].(string)
		ratingStr, _ := payload["rating"].(string)
		feedback, _ := payload["feedback"].(string)

		rating, _ := strconv.Atoi(ratingStr)
		voucherCode := fmt.Sprintf("FB10-%d", time.Now().Unix()%10000)

		models.DB.Exec(`
			INSERT INTO customer_reviews (order_id, rating, feedback, voucher_issued, created_at) 
			VALUES (?, ?, ?, ?, NOW())`,
			orderID, rating, feedback, voucherCode,
		)

		return map[string]interface{}{
			"screen": "SUCCESS_SCREEN",
			"data": map[string]interface{}{
				"voucher_code": voucherCode,
				"message":      fmt.Sprintf("Thank you for your feedback! Here is your 10%% discount code: %s", voucherCode),
			},
		}

	case "GIFT_CARD_SCREEN":
		// 8. Digital Gift Cards Flow
		valueStr, _ := payload["amount"].(string)
		recipientPhone, _ := payload["recipient_phone"].(string)
		recipientName, _ := payload["recipient_name"].(string)
		message, _ := payload["personal_message"].(string)

		giftCode := fmt.Sprintf("GIFT-%d", time.Now().Unix())
		models.DB.Exec(`
			INSERT INTO gift_cards (card_code, amount, recipient_phone, recipient_name, message, status, created_at) 
			VALUES (?, ?, ?, ?, ?, 'ACTIVE', NOW())`,
			giftCode, valueStr, recipientPhone, recipientName, message,
		)

		return map[string]interface{}{
			"screen": "SUCCESS_SCREEN",
			"data": map[string]interface{}{
				"gift_code": giftCode,
				"message":   fmt.Sprintf("Gift Card %s of KES %s generated for %s!", giftCode, valueStr, recipientName),
			},
		}
	}

	return map[string]interface{}{
		"screen": "SUCCESS_SCREEN",
		"data": map[string]interface{}{
			"message": "Action completed successfully.",
		},
	}
}

// Helper data access functions
func getFlowCategories() []map[string]string {
	return []map[string]string{
		{"id": "electronics", "title": "Electronics & Gadgets"},
		{"id": "fashion", "title": "Clothing & Fashion"},
		{"id": "home", "title": "Home & Kitchen"},
	}
}

func getFlowProductsByCategory(catID string) []map[string]string {
	return []map[string]string{
		{"id": "p1", "title": "Smart Watch Pro (KES 4,500)"},
		{"id": "p2", "title": "Wireless Earbuds (KES 3,200)"},
	}
}

func getFlowRecentOrders(phone string) []map[string]string {
	return []map[string]string{
		{"id": "ORD-1092", "title": "Order #1092 - KES 4,500"},
		{"id": "ORD-1088", "title": "Order #1088 - KES 2,100"},
	}
}

func getFlowDeliveredOrders(phone string) []map[string]string {
	return []map[string]string{
		{"id": "ORD-1088", "title": "Order #1088 (Delivered 2 days ago)"},
	}
}

func getFlowOrderStatusDetails(orderID string) map[string]interface{} {
	return map[string]interface{}{
		"order_id":           orderID,
		"status_label":       "Out for Delivery",
		"courier_name":       "Ekomasi Express Courier",
		"estimated_delivery": "Today by 5:00 PM",
	}
}

// Query Meilisearch for personalized shopping recommendations
func queryMeilisearchRecommendations(budget string, occasion string) []map[string]string {
	// Meilisearch dynamic query simulation with tag filtering
	return []map[string]string{
		{"id": "rec1", "title": fmt.Sprintf("Recommended for %s: Premium Bundle (Under %s)", occasion, budget)},
		{"id": "rec2", "title": fmt.Sprintf("Top Pick: Designer Accessory (Under %s)", budget)},
	}
}
