package realtime

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"github.com/gorilla/websocket"
)

func (h *Hub) ServeWebSocket(w http.ResponseWriter, r *http.Request) {
	 sessionID := r.URL.Query().Get("sessionId")
    userType := r.URL.Query().Get("userType")
    userID := r.URL.Query().Get("userId")

    // Validate required parameters
    if sessionID == "" {
        http.Error(w, "sessionId is required", http.StatusBadRequest)
        return
    }
    if userType == "" {
        http.Error(w, "userType is required", http.StatusBadRequest)
        return
    }
    if userID == "" {
        http.Error(w, "userId is required", http.StatusBadRequest)
        return
    }

	// Validate userType
    if userType != "organizer" && userType != "participant" {
        http.Error(w, "userType must be 'organizer' or 'participant'", http.StatusBadRequest)
        return
    }


	conn, err := upgrader.Upgrade(w, r, nil);
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
        return;
	}

	client := &Client {
		hub: h,
		conn : conn,
		send: make(chan []byte, 256),
		sessionID: sessionID,
		userType: userType,
		userID: userID,
	}

	// register client
	h.register <- client;
 log.Printf("WebSocket connection established: session=%s, userType=%s, userID=%s", 
        sessionID, userType, userID)

    // Start goroutines for handling connection
    go client.writePump()
    go client.readPump()

}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close();
		log.Printf("Read pump stopped: session=%s", c.sessionID)
	}()

	// configure connection.
	c.conn.SetReadLimit(512);
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second));
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil;
	})


	for {
		_, message , err := c.conn.ReadMessage();
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket read error: %v", err)
            }
            break
		}

		c.handleMessage(message);
	}
}


func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second) // ping interval.
	defer func(){
		ticker.Stop();
		c.conn.Close();
		log.Printf("Write pump stopped: session=%s", c.sessionID);
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second));


			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{});
				return;
			}


			// write message to websocket.
			writer, err := c.conn.NextWriter(websocket.TextMessage);
			if err != nil {
				return;
			}

			writer.Write(message);


			// close the writer.
			if err := writer.Close(); err != nil {
				return;
			}

		case <-ticker.C :
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second));
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
		}
	}
}


func (c *Client) handleMessage(message []byte) {
	var msg map[string]interface{};
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to unmarshal WebSocket message: %v", err)
        return
	}

	messageType, ok := msg["type"].(string);
	if !ok {
		log.Printf("Message missing type field")
        return
	}


	switch messageType {
	case "ping":
		response := map[string]string{"type":"pong"}
		responseJSON,_ := json.Marshal(response);

		c.send <- responseJSON
		log.Printf("Responded to ping: session=%s", c.sessionID)

	case "subscribe":
		response := map[string]interface{}{
			"type":"subscribed",
			"sessionId":c.sessionID,
			"timestamp":time.Now(),
		}

		responseJSON, _ := json.Marshal(response);
		c.send <- responseJSON;
		log.Printf("Client subscribed: session=%s, userType=%s", c.sessionID, c.userType)


	case "get_results":
		c.handleGetResults();

	default:
        log.Printf("Unknown message type: %s", messageType)
        // Send error response
        errorMsg := map[string]interface{}{
            "type": "error",
            "message": "Unknown message type",
        }
        errorJSON, _ := json.Marshal(errorMsg)
        c.send <- errorJSON
	}
}


// handleGetResults sends current results to client (placeholder for now)
func (c *Client) handleGetResults() {
    // This will be implemented when we have results retrieval
    response := map[string]interface{}{
        "type": "results",
        "message": "Results retrieval coming soon",
        "sessionId": c.sessionID,
    }
    responseJSON, _ := json.Marshal(response)
    c.send <- responseJSON
}

// BroadcastToSession sends a message to all clients in a session
func (h *Hub) BroadcastToSession(sessionID string, message []byte) {
    broadcastMsg := &BroadcastMessage{
        SessionID: sessionID,
        Data:      message,
    }
    h.broadcast <- broadcastMsg
}

// GetSessionStats returns statistics about a session's connections
func (h *Hub) GetSessionStats(sessionID string) map[string]interface{} {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    sessionHub, exists := h.sessions[sessionID]
    if !exists {
        return map[string]interface{}{
            "sessionId": sessionID,
            "activeConnections": 0,
            "exists": false,
        }
    }

    sessionHub.mutex.RLock()
    defer sessionHub.mutex.RUnlock()

    organizerCount := 0
    participantCount := 0

    for client := range sessionHub.clients {
        if client.userType == "organizer" {
            organizerCount++
        } else {
            participantCount++
        }
    }

    return map[string]interface{}{
        "sessionId": sessionID,
        "activeConnections": len(sessionHub.clients),
        "organizers": organizerCount,
        "participants": participantCount,
        "exists": true,
    }
}