package realtime

import (
	"log"
)

func (h *Hub) Run() {
	log.Println("WebSocket Hub started running...");

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToSession(message.SessionID, message.Data);
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock();
	defer h.mutex.Unlock();


	// create session hub.
	sessionHub, exists := h.sessions[client.sessionID];
	if !exists {
		sessionHub = &SessionHub{
			clients: make(map[*Client]bool),
			broadcast: make(chan []byte, 256),
			register: make(chan *Client),
			unregister: make(chan *Client),
		}

		h.sessions[client.sessionID] = sessionHub;
		go sessionHub.run();
	}

	// Register client with session hub
    sessionHub.register <- client

    log.Printf("Client registered: session=%s, type=%s, total_clients=%d", 
        client.sessionID, client.userType, len(sessionHub.clients))
}


func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock();
	defer h.mutex.Unlock();

	if sessionHub, exists := h.sessions[client.sessionID]; exists {
		sessionHub.unregister <- client;

		if len(sessionHub.clients) == 0 {
			delete(h.sessions, client.sessionID);
			log.Printf("Session hub cleaned up: %s", client.sessionID);
		}
	}


	close(client.send);
	log.Printf("Client unregistered: session=%s", client.sessionID);
}

func (h *Hub) broadcastToSession(sessionID string, message []byte) {
    h.mutex.RLock()
    defer h.mutex.RUnlock()

    if sessionHub, exists := h.sessions[sessionID]; exists {
        sessionHub.broadcast <- message
    }
}