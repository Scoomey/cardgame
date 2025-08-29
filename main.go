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
	Card *Card  // card currently played
	Attr string // chosen attribute
}

// GameRoom represents a game between 2 players
type GameRoom struct {
	ID      string
	Players []*Player
	Mutex   sync.Mutex
	Round   int
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
				// Start game: deal half deck each
				mid := len(Deck) / 2
				room.Players[0].Deck = append([]Card{}, Deck[:mid]...)
				room.Players[1].Deck = append([]Card{}, Deck[mid:]...)

				for _, p := range room.Players {
					p.Conn.WriteJSON(map[string]interface{}{
						"action": "start",
						"deck":   p.Deck,
					})
				}
			}

		case "playCard":
			if currentRoom == nil {
				continue
			}

			cardMap := msg["card"].(map[string]interface{})
			attr := msg["attribute"].(string)

			// find card in player's deck (assume always top card for MVP)
			var chosenCard *Card
			for i := range currentPlayer.Deck {
				if currentPlayer.Deck[i].Name == cardMap["name"].(string) {
					chosenCard = &currentPlayer.Deck[i]
					break
				}
			}

			if chosenCard == nil {
				log.Println("Card not found in player's deck")
				continue
			}

			currentPlayer.Card = chosenCard
			currentPlayer.Attr = attr

			// check if both players have played
			if len(currentRoom.Players) == 2 {
				p1 := currentRoom.Players[0]
				p2 := currentRoom.Players[1]

				if p1.Card != nil && p2.Card != nil {
					resolveRound(currentRoom, p1, p2)
					// reset round
					p1.Card, p1.Attr = nil, ""
					p2.Card, p2.Attr = nil, ""
					currentRoom.Round++
				}
			}
		}
	}
}

func resolveRound(room *GameRoom, p1, p2 *Player) {
	attr := p1.Attr // assume same attr chosen for MVP
	val1 := p1.Card.Stats[attr]
	val2 := p2.Card.Stats[attr]

	var winner *Player
	if val1 > val2 {
		winner = p1
	} else if val2 > val1 {
		winner = p2
	}

	// Winner takes both cards (append to bottom of deck, remove from losers)
	if winner != nil {
		winner.Deck = append(winner.Deck, *p1.Card, *p2.Card)

		// remove top cards (simplest way — rebuild slice)
		p1.Deck = removeCard(p1.Deck, p1.Card.Name)
		p2.Deck = removeCard(p2.Deck, p2.Card.Name)
	} else {
		// draw → cards go back to each deck’s bottom
		p1.Deck = append(p1.Deck, *p1.Card)
		p1.Deck = removeCard(p1.Deck, p1.Card.Name)
		p2.Deck = append(p2.Deck, *p2.Card)
		p2.Deck = removeCard(p2.Deck, p2.Card.Name)
	}

	// Check game over
	var gameOver string
	if len(p1.Deck) == 0 {
		gameOver = p2.ID
	} else if len(p2.Deck) == 0 {
		gameOver = p1.ID
	}

	// Notify both players
	for _, p := range room.Players {
		result := map[string]interface{}{
			"action": "roundResult",
			"round":  room.Round,
			"p1Card": p1.Card,
			"p2Card": p2.Card,
			"attr":   attr,
			"winner": func() string {
				if winner != nil {
					return winner.ID
				} else {
					return "draw"
				}
			}(),
			"p1Deck":   len(p1.Deck),
			"p2Deck":   len(p2.Deck),
			"gameOver": gameOver,
		}
		p.Conn.WriteJSON(result)
	}
}

func removeCard(deck []Card, name string) []Card {
	newDeck := []Card{}
	removed := false
	for _, c := range deck {
		if c.Name == name && !removed {
			removed = true
			continue
		}
		newDeck = append(newDeck, c)
	}
	return newDeck
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	fmt.Println("✅ Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
