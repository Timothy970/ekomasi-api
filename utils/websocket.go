// Package utils provides WebSocket utilities for real-time communication in the Adenzo e-commerce platform.
package utils

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// upgrader configures WebSocket upgrade from HTTP.
var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true }, // Allow all origins
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Connection storage map with mutex protection
var (
	activeConnections = make(map[string]*websocket.Conn) // Key -> WebSocket connection
	mu                sync.RWMutex                       // Protects concurrent map access
)

// WebSocket timing and size constants
const (
	writeWait      = 10 * time.Second    // Maximum time to write message
	pongWait       = 60 * time.Second    // Maximum time to wait for pong response
	pingPeriod     = (pongWait * 9) / 10 // Send pings at 90% of pong wait time
	maxMessageSize = 512                 // Maximum message size in bytes
)

// HandleWebSocket upgrades HTTP connection to WebSocket and manages lifecycle.
// It uses a "key" query parameter to identify the connection.
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract connection identifier from query parameter
	key := r.URL.Query().Get("key")
	if key == "" {
		// Fallback for backward compatibility or if specific identifiers are provided
		orderID := r.URL.Query().Get("order_id")
		deliveryID := r.URL.Query().Get("delivery_id")
		if orderID != "" && deliveryID != "" {
			key = orderID + ":" + deliveryID
		} else {
			key = r.URL.Query().Get("user_id")
		}
	}

	if key == "" {
		http.Error(w, "Missing connection identifier (key, user_id, or order_id+delivery_id)", http.StatusBadRequest)
		return
	}

	log.Printf("WebSocket connection request for key: %s", key)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// Configure connection settings
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Store connection in map with thread safety
	mu.Lock()
	activeConnections[key] = conn
	log.Printf("WebSocket connected for key %s", key)
	mu.Unlock()

	// Setup ping ticker for keep-alive
	ticker := time.NewTicker(pingPeriod)
	// Cleanup function when connection closes
	defer func() {
		ticker.Stop()
		conn.Close()

		// Remove connection from map
		mu.Lock()
		delete(activeConnections, key)
		log.Printf("WebSocket closed for key %s", key)
		mu.Unlock()
	}()

	// Read pump: Keep connection alive by reading messages
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error for %s: %v", key, err)
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

// SendToUser sends a JSON message to a specific connection identified by key.
func SendToUser(key string, message interface{}) {
	var conn *websocket.Conn
	var ok bool

	// Look up connection with read lock
	mu.RLock()
	conn, ok = activeConnections[key]
	mu.RUnlock()

	// Connection not found - log and return
	if !ok || conn == nil {
		log.Printf("No WebSocket connection found for key=%s", key)
		return
	}

	// Set write deadline to prevent hanging
	conn.SetWriteDeadline(time.Now().Add(writeWait))
	// Send JSON message
	if err := conn.WriteJSON(message); err != nil {
		log.Printf("WebSocket send error for key %s: %v", key, err)
		conn.Close()

		// Remove failed connection from map
		mu.Lock()
		delete(activeConnections, key)
		mu.Unlock()
	}
}

// IsConnected checks if a key has an active WebSocket connection.
func IsConnected(key string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := activeConnections[key]
	return ok
}

// GetConnectedCount returns total number of active WebSocket connections.
func GetConnectedCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(activeConnections)
}
