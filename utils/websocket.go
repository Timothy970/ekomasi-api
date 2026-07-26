// Package utils provides WebSocket utilities for real-time communication in the Ekomasi e-commerce platform.
package utils

import (
	"encoding/json"
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

// WSMessage defines the structure for Redis-broadcasted WebSocket messages.
type WSMessage struct {
	Key     string      `json:"key"`
	Payload interface{} `json:"payload"`
}

// wsChannel is the Redis channel name for WebSocket broadcasts.
const wsChannel = "ws_broadcast"

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

		// Remove connection from map ONLY if it's the same connection we added
		// This prevents a newer reconnection from being deleted by an old connection's cleanup
		mu.Lock()
		if activeConnections[key] == conn {
			delete(activeConnections, key)
			log.Printf("WebSocket closed for key %s", key)
		} else {
			log.Printf("WebSocket for key %s was already replaced by a newer connection, skipping cleanup", key)
		}
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

// SendToUser broadcasts a JSON message to a specific connection identified by key.
// In a multi-instance environment, this publishes to Redis so all instances can check local connections.
func SendToUser(key string, message interface{}) {
	if RedisClient == nil {
		// Fallback to local send if Redis is not configured
		sendLocally(key, message)
		return
	}

	// Prepare the broadcast message
	broadcastMsg := WSMessage{
		Key:     key,
		Payload: message,
	}

	data, err := json.Marshal(broadcastMsg)
	if err != nil {
		log.Printf("Error marshaling WebSocket broadcast message: %v", err)
		return
	}

	// Publish to Redis
	err = RedisClient.Publish(ctx, wsChannel, data).Err()
	if err != nil {
		log.Printf("Error publishing WebSocket message to Redis: %v", err)
		// Fallback to local send as last resort
		sendLocally(key, message)
	}
}

// sendLocally sends a JSON message to a connection on the current server instance.
func sendLocally(key string, message interface{}) {
	var conn *websocket.Conn
	var ok bool

	// Look up connection with read lock
	mu.RLock()
	conn, ok = activeConnections[key]
	mu.RUnlock()

	// Connection not found - return without logging as error
	// In multi-pod environments, this is expected behavior on pods where the user is not connected
	if !ok || conn == nil {
		log.Printf("No active WebSocket connection for key %s on this instance", key)
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

// StartWebSocketBroadcaster starts a background listener for Redis WebSocket broadcasts.
// It includes a retry mechanism to handle transient Redis connection failures.
func StartWebSocketBroadcaster() {
	if RedisClient == nil {
		log.Println("RedisClient not initialized. WebSocket broadcaster will not start.")
		return
	}

	go func() {
		for {
			pubsub := RedisClient.Subscribe(ctx, wsChannel)
			
			// Verify subscription is working
			_, err := pubsub.Receive(ctx)
			if err != nil {
				log.Printf("Error subscribing to WebSocket channel: %v. Retrying in 5s...", err)
				pubsub.Close()
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("WebSocket broadcaster started on channel: %s", wsChannel)

			ch := pubsub.Channel()
			for msg := range ch {
				var wsMsg WSMessage
				err := json.Unmarshal([]byte(msg.Payload), &wsMsg)
				if err != nil {
					log.Printf("Error unmarshaling WebSocket broadcast: %v", err)
					continue
				}

				// Deliver to local connection if it exists
				sendLocally(wsMsg.Key, wsMsg.Payload)
			}

			log.Println("WebSocket broadcaster channel closed. Reconnecting...")
			pubsub.Close()
			time.Sleep(1 * time.Second)
		}
	}()
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
