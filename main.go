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

	var currentPlayer *Player
	var currentRoom *GameRoom
	
	defer func() {
		conn.Close()
		// Clean up player when connection closes
		if currentPlayer != nil && currentRoom != nil {
			removePlayerFromRoom(currentRoom, currentPlayer.ID)
		}
	}()

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
			log.Printf("Player %s joining room %s", playerID, roomID)
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
			
			// Check if player is already in the room (re-joining)
			existingPlayerIndex := -1
			for i, p := range room.Players {
				if p.ID == playerID {
					existingPlayerIndex = i
					break
				}
			}
			
			if existingPlayerIndex != -1 {
				// Player is re-joining - replace their connection
				room.Players[existingPlayerIndex] = currentPlayer
				log.Printf("Player %s re-joined room %s (replacing old connection)", playerID, roomID)
				
				// If game was in progress, restart it for the re-joining player
				if room.Round > 0 {
					log.Printf("Restarting game in room %s due to player re-join", roomID)
					startGame(room)
				}
			} else {
				// New player joining
				if len(room.Players) >= 2 {
					log.Printf("Player %s tried to join full room %s", playerID, roomID)
					if err := currentPlayer.Conn.WriteJSON(map[string]interface{}{
						"action":  "error",
						"message": "Room is full",
					}); err != nil {
						log.Println("Error sending error message:", err)
					}
					return
				}
				room.Players = append(room.Players, currentPlayer)
				log.Printf("Player %s joined room %s (new player)", playerID, roomID)
			}
			
			playerCount := len(room.Players)
			currentRoom = room
			room.Mutex.Unlock()

			// Always send response to joining player
			if playerCount == 1 {
				log.Printf("First player in room %s, sending waiting message", roomID)
				if err := currentPlayer.Conn.WriteJSON(map[string]interface{}{
					"action":  "waiting",
					"message": "Waiting for another player to join...",
				}); err != nil {
					log.Println("Error sending waiting message:", err)
				}
			}

			if playerCount == 2 {
				// Start or restart game when we have 2 players
				log.Printf("Second player joined room %s, starting game!", roomID)
				startGame(room)
			}

		case "playCard":
			if currentRoom == nil {
				continue
			}

			// Check if it's this player's turn
			playerIndex := -1
			for i, p := range currentRoom.Players {
				if p.ID == currentPlayer.ID {
					playerIndex = i
					break
				}
			}
			
			if playerIndex == -1 {
				log.Printf("Player %s not found in room", currentPlayer.ID)
				continue
			}
			
			if playerIndex != currentRoom.TurnIndex {
				log.Printf("Player %s tried to play out of turn", currentPlayer.ID)
				continue
			}

			// Only need the attribute, not the card
			attr := msg["attribute"].(string)
			
			// Set the current player's card and attribute
			currentPlayer.Card = &currentPlayer.Deck[0]
			currentPlayer.Attr = attr
			
			log.Printf("Player %s played attribute %s for card %s", currentPlayer.ID, attr, currentPlayer.Card.Name)

			// Resolve the round immediately when current player plays
			if len(currentRoom.Players) == 2 {
				p1 := currentRoom.Players[0]
				p2 := currentRoom.Players[1]
				
				// Automatically resolve the round using both players' top cards
				// The current player's card is already set, now set the opponent's card
				opponentIndex := 1 - playerIndex
				opponent := currentRoom.Players[opponentIndex]
				
				// Set opponent's card (they don't need to choose attribute, use a default or random one)
				opponent.Card = &opponent.Deck[0]
				// For now, use the same attribute as the current player
				opponent.Attr = attr
				
				log.Printf("Resolving round: %s vs %s with attribute %s", 
					currentPlayer.Card.Name, opponent.Card.Name, attr)
				
				// Resolve the round
				resolveRound(currentRoom, p1, p2)

				// Clear played cards and attributes
				p1.Card, p1.Attr = nil, ""
				p2.Card, p2.Attr = nil, ""
				currentRoom.Round++
				
				// Note: TurnIndex will be set by resolveRound based on winner
			}
		}
	}
}

// startGame initializes or restarts a game for a room
func startGame(room *GameRoom) {
	// Reset game state
	room.Round = 1
	room.TurnIndex = rand.Intn(2)
	
	// Clear any existing game state
	for _, p := range room.Players {
		p.Deck = nil
		p.Card = nil
		p.Attr = ""
	}
	
	// Shuffle and deal cards
	shuffled := append([]Card{}, Deck...)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	mid := len(shuffled) / 2
	room.Players[0].Deck = append([]Card{}, shuffled[:mid]...)
	room.Players[1].Deck = append([]Card{}, shuffled[mid:]...)

	// Send start messages to all players
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
	
	log.Printf("Game started/restarted in room %s", room.ID)
}

// removePlayerFromRoom removes a player from a room and handles cleanup
func removePlayerFromRoom(room *GameRoom, playerID string) {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	
	// Find and remove the player
	for i, p := range room.Players {
		if p.ID == playerID {
			room.Players = append(room.Players[:i], room.Players[i+1:]...)
			log.Printf("Player %s removed from room %s", playerID, room.ID)
			break
		}
	}
	
	// If room is empty, remove it entirely
	if len(room.Players) == 0 {
		roomsMutex.Lock()
		delete(rooms, room.ID)
		roomsMutex.Unlock()
		log.Printf("Room %s removed (empty)", room.ID)
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

	// Set the next turn based on the winner
	if winner != nil {
		// Winner gets the next turn
		for i, p := range room.Players {
			if p.ID == winner.ID {
				room.TurnIndex = i
				break
			}
		}
		log.Printf("Player %s won the round and gets the next turn", winner.ID)
	} else {
		// In case of a draw, alternate turns
		room.TurnIndex = 1 - room.TurnIndex
		log.Printf("Round was a draw, turn alternates to player %d", room.TurnIndex)
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
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		roomsMutex.Lock()
		defer roomsMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		status := map[string]interface{}{
			"status":       "running",
			"rooms":        len(rooms),
			"active_rooms": make(map[string]interface{}),
		}

		for roomID, room := range rooms {
			room.Mutex.Lock()
			status["active_rooms"].(map[string]interface{})[roomID] = map[string]interface{}{
				"players": len(room.Players),
				"round":   room.Round,
			}
			room.Mutex.Unlock()
		}

		fmt.Fprintf(w, "%v", status)
	})

	fmt.Println("✅ Server started on :8080")
	fmt.Println("📊 Status endpoint available at /status")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
