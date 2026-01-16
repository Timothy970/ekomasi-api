// Package utils provides WebSocket utilities for real-time communication in the Adenzo e-commerce platform.
//
// This file contains WebSocket connection management:
//   - WebSocket connection upgrade from HTTP
//   - Connection storage and retrieval (users and guests)
//   - Real-time message sending to specific users/guests
//   - Connection health monitoring (ping/pong)
//   - Automatic connection cleanup
//   - Thread-safe concurrent connection management
//
// WebSocket Features:
//   - User connections: Identified by user_id
//   - Guest connections: Identified by order_id + delivery_id
//   - Automatic ping/pong keep-alive mechanism
//   - Read/write deadlines for connection timeouts
//   - Connection state tracking
//   - Graceful connection cleanup on disconnect
//
// Connection Types:
//   - Authenticated users: Tracked by user_id
//   - Guest users: Tracked by order_id:delivery_id composite key
//
// Thread Safety:
//   - All connection maps protected by sync.RWMutex
//   - Safe concurrent read/write operations
package utils

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// upgrader configures WebSocket upgrade from HTTP.
//
// Settings:
//   - CheckOrigin: Allows all origins (CORS permissive)
//   - ReadBufferSize: 1024 bytes for incoming messages
//   - WriteBufferSize: 1024 bytes for outgoing messages
var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true }, // Allow all origins
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Connection storage maps with mutex protection
var (
	userConnections  = make(map[string]*websocket.Conn) // User ID -> WebSocket connection
	guestConnections = make(map[string]*websocket.Conn) // Guest key -> WebSocket connection
	mu               sync.RWMutex                       // Protects concurrent map access
)

// WebSocket timing and size constants
const (
	writeWait      = 10 * time.Second    // Maximum time to write message
	pongWait       = 60 * time.Second    // Maximum time to wait for pong response
	pingPeriod     = (pongWait * 9) / 10 // Send pings at 90% of pong wait time
	maxMessageSize = 512                 // Maximum message size in bytes
)

// makeGuestKey creates a composite key for guest connections.
//
// Combines order ID and delivery ID to uniquely identify guest users
// who don't have authenticated user accounts.
//
// Parameters:
//   - orderID: string - Order identifier
//   - deliveryID: string - Delivery identifier
//
// Returns:
//   - string: Composite key in format "orderID:deliveryID"
func makeGuestKey(orderID, deliveryID string) string {
	return orderID + ":" + deliveryID
}

// HandleWebSocket upgrades HTTP connection to WebSocket and manages lifecycle.
//
// This function:
// 1. Validates connection identifiers (user_id or order_id+delivery_id)
// 2. Upgrades HTTP connection to WebSocket
// 3. Configures connection settings (read limit, deadlines, pong handler)
// 4. Stores connection in appropriate map
// 5. Starts ping/pong keep-alive mechanism
// 6. Handles connection cleanup on disconnect
//
// Connection Identification:
//   - Authenticated users: Provide user_id query parameter
//   - Guest users: Provide order_id + delivery_id query parameters
//
// Query Parameters:
//   - user_id: User identifier (optional if order_id+delivery_id provided)
//   - order_id: Order identifier (required for guests)
//   - delivery_id: Delivery identifier (required for guests)
//
// Parameters:
//   - w: http.ResponseWriter - HTTP response writer
//   - r: *http.Request - HTTP request with query parameters
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract connection identifiers from query parameters
	userID := r.URL.Query().Get("user_id")
	orderID := r.URL.Query().Get("order_id")
	deliveryID := r.URL.Query().Get("delivery_id")

	// Check if writer supports hijacking (required for WebSocket)
	_, ok := w.(http.Hijacker)
	if !ok {
		log.Println("Writer does NOT implement Hijacker:", w)
	}

	// Validate that at least one identification method is provided
	if userID == "" && (orderID == "" || deliveryID == "") {
		http.Error(w, "Missing identifier (user_id or order_id+delivery_id)", http.StatusBadRequest)
		return
	}

	log.Printf("WebSocket connection request: user_id=%s, order_id=%s, delivery_id=%s", userID, orderID, deliveryID)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// Configure connection settings
	conn.SetReadLimit(maxMessageSize)              // Limit message size
	conn.SetReadDeadline(time.Now().Add(pongWait)) // Set initial read deadline
	// Setup pong handler to reset read deadline on pong receipt
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Store connection in appropriate map with thread safety
	mu.Lock()
	if userID != "" {
		// Store authenticated user connection
		userConnections[userID] = conn
		log.Printf("WebSocket connected for user %s", userID)
	} else {
		// Store guest user connection
		key := makeGuestKey(orderID, deliveryID)
		guestConnections[key] = conn
		log.Printf("WebSocket connected for guest %s", key)
	}
	mu.Unlock()

	// Setup ping ticker for keep-alive
	ticker := time.NewTicker(pingPeriod)
	// Cleanup function when connection closes
	defer func() {
		ticker.Stop()
		conn.Close()

		// Remove connection from map
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

	// Read pump: Keep connection alive by reading messages
	// This goroutine exits when connection closes or error occurs
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				// Log unexpected close errors
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}
		}
	}()

	// Ping pump: Send periodic pings to keep connection alive
	for range ticker.C {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return // Exit on ping send failure
		}
	}
}

// SendToUser sends a JSON message to a specific user or guest connection.
//
// This function:
// 1. Looks up connection by user_id or order_id+delivery_id
// 2. Sets write deadline to prevent hanging
// 3. Sends JSON message
// 4. Cleans up connection on error
//
// Connection Lookup Priority:
//   - If userID provided: Look in userConnections
//   - If orderID+deliveryID provided: Look in guestConnections
//
// Parameters:
//   - userID: string - User identifier (empty for guest users)
//   - orderID: string - Order identifier (for guest users)
//   - deliveryID: string - Delivery identifier (for guest users)
//   - message: interface{} - Message to send (will be JSON encoded)
func SendToUser(userID, orderID, deliveryID string, message interface{}) {
	var conn *websocket.Conn
	var ok bool

	// Look up connection with read lock
	mu.RLock()
	if userID != "" {
		// Look up authenticated user connection
		conn, ok = userConnections[userID]
	} else if orderID != "" && deliveryID != "" {
		// Look up guest user connection
		key := makeGuestKey(orderID, deliveryID)
		conn, ok = guestConnections[key]
	}
	mu.RUnlock()

	// Connection not found - log and return
	if !ok || conn == nil {
		log.Printf("No WebSocket connection found for user_id=%s, order_id=%s, delivery_id=%s",
			userID, orderID, deliveryID)
		return
	}

	// Set write deadline to prevent hanging
	conn.SetWriteDeadline(time.Now().Add(writeWait))
	// Send JSON message
	if err := conn.WriteJSON(message); err != nil {
		log.Printf("WebSocket send error: %v", err)
		conn.Close()

		// Remove failed connection from map
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

// IsUserConnected checks if a user has an active WebSocket connection.
//
// Uses read lock for thread-safe concurrent access.
//
// Parameters:
//   - userID: string - User identifier to check
//
// Returns:
//   - bool: true if user is connected, false otherwise
func IsUserConnected(userID string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := userConnections[userID]
	return ok
}

// GetConnectedUserCount returns total number of active WebSocket connections.
//
// Counts both authenticated user connections and guest connections.
// Uses read lock for thread-safe concurrent access.
//
// Returns:
//   - int: Total count of active connections (users + guests)
func GetConnectedUserCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(userConnections) + len(guestConnections)
}
