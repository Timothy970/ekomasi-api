package utils

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	userConnections  = make(map[string]*websocket.Conn)
	guestConnections = make(map[string]*websocket.Conn)
	mu               sync.RWMutex // Protect concurrent map access
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

func makeGuestKey(orderID, deliveryID string) string {
	return orderID + ":" + deliveryID
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	orderID := r.URL.Query().Get("order_id")
	deliveryID := r.URL.Query().Get("delivery_id")
	_, ok := w.(http.Hijacker)
	if !ok {
		log.Println("Writer does NOT implement Hijacker:", w)
	}

	if userID == "" && (orderID == "" || deliveryID == "") {
		http.Error(w, "Missing identifier (user_id or order_id+delivery_id)", http.StatusBadRequest)
		return
	}
	log.Printf("WebSocket connection request: user_id=%s, order_id=%s, delivery_id=%s", userID, orderID, deliveryID)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// Configure connection
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Store connection with thread safety
	mu.Lock()
	if userID != "" {
		userConnections[userID] = conn
		log.Printf("WebSocket connected for user %s", userID)
	} else {
		key := makeGuestKey(orderID, deliveryID)
		guestConnections[key] = conn
		log.Printf("WebSocket connected for guest %s", key)
	}
	mu.Unlock()

	// Start ping ticker
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()

		mu.Lock()
		if userID != "" {
			delete(userConnections, userID)
			log.Printf("WebSocket closed for user %s", userID)
		} else {
			key := makeGuestKey(orderID, deliveryID)
			delete(guestConnections, key)
			log.Printf("WebSocket closed for guest %s", key)
		}
		mu.Unlock()
	}()

	// Read pump (keep connection alive)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}
		}
	}()

	// Ping pump (send periodic pings)
	for range ticker.C {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}

func SendToUser(userID, orderID, deliveryID string, message interface{}) {
	var conn *websocket.Conn
	var ok bool

	mu.RLock()
	if userID != "" {
		conn, ok = userConnections[userID]
	} else if orderID != "" && deliveryID != "" {
		key := makeGuestKey(orderID, deliveryID)
		conn, ok = guestConnections[key]
	}
	mu.RUnlock()

	if !ok || conn == nil {
		log.Printf("No WebSocket connection found for user_id=%s, order_id=%s, delivery_id=%s",
			userID, orderID, deliveryID)
		return
	}

	conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(message); err != nil {
		log.Printf("WebSocket send error: %v", err)
		conn.Close()

		mu.Lock()
		if userID != "" {
			delete(userConnections, userID)
		} else {
			key := makeGuestKey(orderID, deliveryID)
			delete(guestConnections, key)
		}
		mu.Unlock()
	}
}

// Check if user is connected
func IsUserConnected(userID string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := userConnections[userID]
	return ok
}

// Get connected user count
func GetConnectedUserCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(userConnections) + len(guestConnections)
}
