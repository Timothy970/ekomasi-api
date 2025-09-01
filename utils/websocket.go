package utils

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	userConnections  = make(map[string]*websocket.Conn) // user_id -> conn
	guestConnections = make(map[string]*websocket.Conn) // "order_id:delivery_id" -> conn
)

func makeGuestKey(orderID, deliveryID string) string {
	return orderID + ":" + deliveryID
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	orderID := r.URL.Query().Get("order_id")
	deliveryID := r.URL.Query().Get("delivery_id")

	if userID == "" && (orderID == "" || deliveryID == "") {
		http.Error(w, "Missing identifier (user_id or order_id+delivery_id)", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	if userID != "" {
		userConnections[userID] = conn
		log.Printf("WebSocket connected for user %s", userID)
	} else {
		key := makeGuestKey(orderID, deliveryID)
		guestConnections[key] = conn
		log.Printf("WebSocket connected for guest %s", key)
	}

	defer func() {
		conn.Close()
		if userID != "" {
			delete(userConnections, userID)
			log.Printf("WebSocket closed for user %s", userID)
		} else {
			key := makeGuestKey(orderID, deliveryID)
			delete(guestConnections, key)
			log.Printf("WebSocket closed for guest %s", key)
		}
	}()

	for {
		_, _, err := conn.NextReader()
		if err != nil {
			break
		}
	}
}

func SendToUser(userID, orderID, deliveryID string, message interface{}) {
	var conn *websocket.Conn
	var ok bool

	if userID != "" {
		conn, ok = userConnections[userID]
	} else if orderID != "" && deliveryID != "" {
		key := makeGuestKey(orderID, deliveryID)
		conn, ok = guestConnections[key]
	}

	if !ok || conn == nil {
		log.Printf("No WebSocket connection found for user_id=%s, order_id=%s, delivery_id=%s",
			userID, orderID, deliveryID)
		return
	}

	if err := conn.WriteJSON(message); err != nil {
		log.Printf("WebSocket send error: %v", err)
		conn.Close()
		if userID != "" {
			delete(userConnections, userID)
		} else {
			key := makeGuestKey(orderID, deliveryID)
			delete(guestConnections, key)
		}
	}
}
