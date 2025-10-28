package realtime

import (
	"log"
)

func (sh *SessionHub) run() {
	for {
		select {
		case client := <-sh.register:
			sh.mutex.Lock();
			sh.clients[client] = true;
			sh.mutex.Unlock();
			log.Printf("Client added to session hub: %s", client.sessionID);

		case client := <-sh.unregister:
			sh.mutex.Lock();
			if _, exists := sh.clients[client]; exists {
				delete(sh.clients, client);
				log.Printf("Client removed from session hub: %s", client.sessionID);
			}

			sh.mutex.Unlock();

		case message := <-sh.broadcast:
			sh.broadcastMessage(message);
		}
	}
}

func (sh *SessionHub) broadcastMessage(message []byte) {
	sh.mutex.Lock();
	defer sh.mutex.Unlock();

	for client := range sh.clients {
		select {
		case client.send <- message:
			// message successfully  sent to client channel.

		default : 
		close(client.send);
		delete(sh.clients, client);
		log.Printf("Client disconnected (slow) : %s", client.sessionID);
		}
	}
}