package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"     // Gin framework for handling HTTP requests
	"github.com/gorilla/websocket" // WebSocket package for real-time communication
)

// Upgrader to convert an HTTP connection into a WebSocket connection
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins (for testing)
}

// Map to store connected WebSocket clients
var clients = make(map[*websocket.Conn]bool)

// Channel to broadcast messages to all clients
var broadcast = make(chan string)

// List of banned words that should be filtered from messages
var bannedWords = []string{"damn", "crap", "stupid"}

// filterMessage checks if a message contains banned words and replaces them with a warning
func filterMessage(msg string) string {
	lowerMsg := strings.ToLower(msg) // Convert message to lowercase for case-insensitive comparison
	for _, word := range bannedWords {
		if strings.Contains(lowerMsg, word) {
			return "*** Warning: Inappropriate language detected ***"
		}
	}
	return msg // Return original message if no banned words are found
}

// handleWebSocket upgrades the HTTP connection to a WebSocket and manages incoming messages
func handleWebSocket(c *gin.Context) {
	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close() // Ensure the connection is closed when the function exits

	// Add client to the clients map
	clients[conn] = true
	fmt.Printf("New client connected: %v | Total clients: %d\n", conn.RemoteAddr(), len(clients))

	// Keep listening for messages from this WebSocket connection
	for {
		_, msg, err := conn.ReadMessage() // Read incoming message
		if err != nil {
			fmt.Printf("Client disconnected: %v | Total clients: %d\n", conn.RemoteAddr(), len(clients)-1)
			delete(clients, conn) // Remove client from map
			break
		}

		// Send received message to the broadcast channel
		broadcast <- string(msg)
	}
}

// broadcastMessages continuously listens for messages and sends them to all connected clients
func broadcastMessages() {
	for {
		msg := <-broadcast                // Wait for a new message from the broadcast channel
		filteredMsg := filterMessage(msg) // Apply message filtering

		// Send the filtered message to all connected clients
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, []byte(filteredMsg))
			if err != nil {
				fmt.Println("Error writing message:", err)
				client.Close()
				delete(clients, client) // Remove unresponsive client
				fmt.Printf("Client removed due to error: %v | Total clients: %d\n", client.RemoteAddr(), len(clients))
			}
		}
	}
}

func main() {
	// Create a new Gin router
	r := gin.Default()

	// Define WebSocket route
	r.GET("/ws", handleWebSocket)

	// Start a goroutine to handle message broadcasting
	go broadcastMessages()

	// Start the Gin HTTP server on port 8080
	fmt.Println("WebSocket server running on :8080")
	r.Run(":8080")
}
