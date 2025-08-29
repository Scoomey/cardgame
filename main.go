package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Player represents a connected player
type Player struct {
	ID   string
	Conn *websocket.Conn
	Deck []Card
}

// GameRoom represents a game between 2 players
type GameRoom struct {
	ID      string
	Players []*Player
	Mutex   sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var rooms = make(map[string]*GameRoom)
var roomsMutex = sync.Mutex{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	defer conn.Close()

	var currentPlayer *Player
	var currentRoom *GameRoom

	// Listen for messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		switch msg["action"] {
		case "join":
			roomID := msg["room"].(string)
			playerID := msg["player"].(string)
			currentPlayer = &Player{ID: playerID, Conn: conn}

			roomsMutex.Lock()
			room, exists := rooms[roomID]
			if !exists {
				room = &GameRoom{ID: roomID}
				rooms[roomID] = room
			}
			roomsMutex.Unlock()

			room.Mutex.Lock()
			room.Players = append(room.Players, currentPlayer)
			currentRoom = room
			room.Mutex.Unlock()

			if len(room.Players) == 2 {
				// Start game: send each player their deck
				for _, p := range room.Players {
					p.Deck = Deck // simple, same deck for both
					p.Conn.WriteJSON(map[string]interface{}{
						"action": "start",
						"deck":   p.Deck,
					})
				}
			}

		case "playCard":
			// Broadcast the played card to the opponent
			card := msg["card"]
			attr := msg["attribute"]
			if currentRoom != nil {
				for _, p := range currentRoom.Players {
					if p.Conn != conn {
						p.Conn.WriteJSON(map[string]interface{}{
							"action":    "opponentPlayed",
							"card":      card,
							"attribute": attr,
						})
					}
				}
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)

	fmt.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
