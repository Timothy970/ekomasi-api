package utils

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Map of user UUID to WebSocket connection
var userConnections = make(map[string]*websocket.Conn)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	userConnections[userID] = conn
	log.Printf("WebSocket connected for user %s", userID)

	// Keep the connection alive
	defer func() {
		conn.Close()
		delete(userConnections, userID)
		log.Printf("WebSocket closed for user %s", userID)
	}()

	for {
		_, _, err := conn.NextReader()
		if err != nil {
			break // connection closed or error
		}
	}
}
func SendToUser(userID string, message interface{}) {
	conn, ok := userConnections[userID]
	if !ok {
		log.Printf("No WebSocket connection for user %s", userID)
		return
	}

	err := conn.WriteJSON(message)
	if err != nil {
		log.Printf("WebSocket send error to user %s: %v", userID, err)
		conn.Close()
		delete(userConnections, userID)
	}
}
