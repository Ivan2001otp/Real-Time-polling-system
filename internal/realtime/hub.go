package realtime

import (
    "sync"
	"net/http"
    "github.com/gorilla/websocket"
)


var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true;
		},
	}
)

type Hub struct {
	sessions map[string]*SessionHub;
	register chan *Client;
	unregister chan *Client;
	broadcast chan *BroadcastMessage;
	mutex sync.RWMutex;
}

type SessionHub struct {
	clients map[*Client]bool;
	broadcast chan []byte;
	register chan *Client;
	unregister chan *Client;
	mutex sync.RWMutex;
}

type Client struct {
	hub *Hub;
	conn *websocket.Conn;
	send chan []byte;
	sessionID string;
	userType string; // "organizer" or "participant"
	userID string;
}


type BroadcastMessage struct {
	SessionID string;
	Data []byte;
}

func NewHub() *Hub {
	return &Hub{
		sessions: make(map[string]*SessionHub),
		register: make(chan *Client),
		unregister: make(chan *Client),
		broadcast: make(chan *BroadcastMessage),
	}
}