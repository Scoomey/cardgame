package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Player represents a connected player
type Player struct {
	ID   string
	Conn *websocket.Conn
	Deck []Card
	Card *Card
	Attr string
}

// Card represents a mythical creature card
type Card struct {
	Name  string
	Stats map[string]int
	Image string
}

// GameRoom represents a game between 2 players
type GameRoom struct {
	ID        string
	Players   []*Player
	Mutex     sync.Mutex
	Round     int
	TurnIndex int // index of player whose turn it is
}

// Sample exotic deck
var Deck = []Card{
	{"Dragon", map[string]int{"Strength": 95, "Speed": 60, "Magic": 90}, "dragon.png"},
	{"Phoenix", map[string]int{"Strength": 70, "Speed": 80, "Magic": 95}, "phoenix.png"},
	{"Unicorn", map[string]int{"Strength": 65, "Speed": 75, "Magic": 85}, "unicorn.png"},
	{"Griffin", map[string]int{"Strength": 80, "Speed": 70, "Magic": 60}, "griffin.png"},
	{"Kraken", map[string]int{"Strength": 85, "Speed": 50, "Magic": 80}, "kraken.png"},
	{"Chimera", map[string]int{"Strength": 75, "Speed": 65, "Magic": 70}, "chimera.png"},
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
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Read error:", err)
			break
		}

		switch msg["action"] {
		case "join":
			roomID := msg["room"].(string)
			playerID := msg["player"].(string)
			currentPlayer = &Player{ID: playerID, Conn: conn}

			// Get or create room
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

			// Always send response to joining player
			if len(room.Players) == 1 {
				if err := currentPlayer.Conn.WriteJSON(map[string]interface{}{
					"action":  "waiting",
					"message": "Waiting for another player to join...",
				}); err != nil {
					log.Println("Error sending waiting message:", err)
				}
			}

			if len(room.Players) == 2 {
				// Second player joins — start game
				shuffled := append([]Card{}, Deck...)
				rand.Seed(time.Now().UnixNano())
				rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

				mid := len(shuffled) / 2
				room.Players[0].Deck = append([]Card{}, shuffled[:mid]...)
				room.Players[1].Deck = append([]Card{}, shuffled[mid:]...)

				room.Round = 1
				room.TurnIndex = rand.Intn(2)

				for i, p := range room.Players {
					opponent := room.Players[1-i]
					if err := p.Conn.WriteJSON(map[string]interface{}{
						"action":           "start",
						"yourTopCard":      p.Deck[0],
						"opponentTopCard":  opponent.Deck[0],
						"yourDeckSize":     len(p.Deck),
						"opponentDeckSize": len(opponent.Deck),
						"yourTurn":         i == room.TurnIndex,
					}); err != nil {
						log.Println("Error sending start message:", err)
					}
				}
			}

		case "playCard":
			if currentRoom == nil {
				continue
			}

			cardMap := msg["card"].(map[string]interface{})
			attr := msg["attribute"].(string)

			// Ensure player plays top card
			if currentPlayer.Deck[0].Name != cardMap["name"].(string) {
				log.Println("Played card does not match top card")
				continue
			}

			currentPlayer.Card = &currentPlayer.Deck[0]
			currentPlayer.Attr = attr

			if len(currentRoom.Players) == 2 {
				p1 := currentRoom.Players[0]
				p2 := currentRoom.Players[1]

				if p1.Card != nil && p2.Card != nil {
					resolveRound(currentRoom, p1, p2)

					p1.Card, p1.Attr = nil, ""
					p2.Card, p2.Attr = nil, ""
					currentRoom.Round++
					currentRoom.TurnIndex = 1 - currentRoom.TurnIndex
				}
			}
		}
	}
}

func resolveRound(room *GameRoom, p1, p2 *Player) {
	attr := p1.Attr
	val1 := p1.Card.Stats[attr]
	val2 := p2.Card.Stats[attr]

	var winner *Player
	if val1 > val2 {
		winner = p1
	} else if val2 > val1 {
		winner = p2
	}

	if winner != nil {
		winner.Deck = append(winner.Deck, *p1.Card, *p2.Card)
	}

	// Remove top cards
	p1.Deck = p1.Deck[1:]
	p2.Deck = p2.Deck[1:]

	var gameOver string
	if len(p1.Deck) == 0 {
		gameOver = p2.ID
	} else if len(p2.Deck) == 0 {
		gameOver = p1.ID
	}

	for i, p := range room.Players {
		opponent := room.Players[1-i]
		var topCard, opTopCard *Card
		if len(p.Deck) > 0 {
			topCard = &p.Deck[0]
		}
		if len(opponent.Deck) > 0 {
			opTopCard = &opponent.Deck[0]
		}

		result := map[string]interface{}{
			"action":           "roundResult",
			"yourTopCard":      topCard,
			"opponentTopCard":  opTopCard,
			"yourDeckSize":     len(p.Deck),
			"opponentDeckSize": len(opponent.Deck),
			"winner": func() string {
				if winner != nil {
					return winner.ID
				}
				return "draw"
			}(),
			"gameOver": gameOver,
			"yourTurn": i == room.TurnIndex,
		}

		if err := p.Conn.WriteJSON(result); err != nil {
			log.Println("Error sending round result:", err)
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	fmt.Println("✅ Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
