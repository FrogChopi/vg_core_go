package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	ID     string
	Conn   *websocket.Conn
	RoomID string
	// Add channel to Client for Mulligan response
	MulliganCh chan []int
	OrderCh    chan string
}

type Room struct {
	ID      string
	Clients map[string]*Client
	Party   *Party
	Mutex   sync.Mutex
}

var (
	rooms     = make(map[string]*Room)
	roomsLock sync.RWMutex
)

func StartServer(port string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("Server started on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	clientID := r.URL.Query().Get("id")
	if clientID == "" {
		clientID = uuid.New().String()
	}

	client := &Client{ID: clientID, Conn: conn, MulliganCh: make(chan []int), OrderCh: make(chan string)}
	defer func() {
		handleQuitRoom(client)
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		action, _ := msg["action"].(string)
		payload, _ := msg["payload"].(map[string]interface{})

		switch action {
		case "create_room":
			roomID, _ := payload["room_id"].(string)
			handleCreateRoom(client, roomID)
		case "join_room":
			if roomID, ok := payload["room_id"].(string); ok {
				handleJoinRoom(client, roomID)
			}
		case "quit_room":
			handleQuitRoom(client)
		case "create_party":
			handleCreateParty(client)
		case "close_party":
			handleCloseParty(client)
		case "mulligan_response":
			if indicesInt, ok := payload["indices"].([]interface{}); ok {
				indices := []int{}
				for _, v := range indicesInt {
					if f, ok := v.(float64); ok {
						indices = append(indices, int(f))
					}
				}
				select {
				case client.MulliganCh <- indices:
				default:
				}
			}
		case "order_response":
			if choice, ok := payload["choice"].(string); ok {
				select {
				case client.OrderCh <- choice:
				default:
				}
			}
		}
	}
}

func handleCreateRoom(client *Client, roomID string) {
	handleQuitRoom(client)

	if roomID == "" {
		roomID = uuid.New().String()
	}
	room := &Room{
		ID:      roomID,
		Clients: make(map[string]*Client),
	}

	roomsLock.Lock()
	rooms[roomID] = room
	roomsLock.Unlock()

	handleJoinRoom(client, roomID)
}

func handleJoinRoom(client *Client, roomID string) {
	handleQuitRoom(client)

	roomsLock.RLock()
	room, exists := rooms[roomID]
	roomsLock.RUnlock()

	if !exists {
		client.Conn.WriteJSON(map[string]string{"error": "Room not found"})
		return
	}

	room.Mutex.Lock()
	room.Clients[client.ID] = client
	client.RoomID = roomID
	playerCount := len(room.Clients)
	room.Mutex.Unlock()

	client.Conn.WriteJSON(map[string]interface{}{"event": "room_joined", "room_id": roomID})
	broadcast(room, map[string]interface{}{"event": "player_joined", "player_count": playerCount, "player_id": client.ID})
}

func handleQuitRoom(client *Client) {
	if client.RoomID == "" {
		return
	}
	roomsLock.RLock()
	room, exists := rooms[client.RoomID]
	roomsLock.RUnlock()
	if exists {
		room.Mutex.Lock()
		delete(room.Clients, client.ID)
		count := len(room.Clients)
		room.Mutex.Unlock()
		if count == 0 {
			roomsLock.Lock()
			delete(rooms, client.RoomID)
			roomsLock.Unlock()
		} else {
			broadcast(room, map[string]interface{}{"event": "player_left", "player_count": count, "player_id": client.ID})
		}
	}
	client.RoomID = ""
}

func handleCreateParty(client *Client) {
	roomsLock.RLock()
	room, exists := rooms[client.RoomID]
	roomsLock.RUnlock()
	if !exists {
		return
	}
	room.Mutex.Lock()
	// defer room.Mutex.Unlock() // Don't defer unlock here if we are going async, need careful locking

	if len(room.Clients) < 2 {
		room.Mutex.Unlock()
		client.Conn.WriteJSON(map[string]string{"error": "Need at least 2 players"})
		return
	}

	// Make a copy of clients for the party to avoid lock issues during async execution
	clientsList := []*Client{}
	for _, c := range room.Clients {
		clientsList = append(clientsList, c)
	}
	room.Mutex.Unlock()

	go func() {
		deck1, err1 := ParseDeckFile("decks/KT_Starter.md")
		deck2, err2 := ParseDeckFile("decks/LM_Starter.md")
		if err1 != nil || err2 != nil {
			client.Conn.WriteJSON(map[string]string{"error": "Failed to load decks"})
			return
		}

		party := InitParty([]*Deck{deck1, deck2})
		InitGame(party, "")

		// Set party on room
		room.Mutex.Lock()
		room.Party = party
		room.Mutex.Unlock()

		broadcast(room, map[string]interface{}{"event": "party_created"})

		// 1. Decide Turn Order
		swapped := party.DecideTurnOrder(
			func(r0, r1 int) {
				// Send roll results
				for i, c := range clientsList {
					c.Conn.WriteJSON(map[string]interface{}{
						"event":      "dice_roll",
						"rolls":      []int{r0, r1},
						"your_index": i,
					})
				}
				time.Sleep(2 * time.Second)
			},
			func(winnerIndex int) string {
				if winnerIndex >= len(clientsList) {
					return "first"
				}
				winnerClient := clientsList[winnerIndex]
				winnerClient.Conn.WriteJSON(map[string]interface{}{"event": "ask_first_second"})

				choice := <-winnerClient.OrderCh
				return choice
			},
		)

		if swapped {
			clientsList[0], clientsList[1] = clientsList[1], clientsList[0]
		}

		// Notify Turn Order
		clientsList[0].Conn.WriteJSON(map[string]string{"event": "turn_order", "msg": "You are going FIRST"})
		clientsList[1].Conn.WriteJSON(map[string]string{"event": "turn_order", "msg": "You are going SECOND"})
		time.Sleep(1 * time.Second)

		// 2. Perform Mulligan (Parallel)
		party.PerformMulligan(func(playerIndex int, hand []*Card) []int {
			if playerIndex >= len(clientsList) {
				return []int{}
			}
			targetClient := clientsList[playerIndex]

			// Send request to client
			handData := []string{}
			for _, c := range hand {
				handData = append(handData, ToString(c))
			}

			targetClient.Conn.WriteJSON(map[string]interface{}{
				"event": "request_mulligan",
				"hand":  handData,
			})

			// Wait for response
			indices := <-targetClient.MulliganCh
			return indices
		})

		// 3. Send Updated Hands
		for i, c := range clientsList {
			player := &party.Players[i]
			handData := []string{}
			for _, card := range player.Hand {
				handData = append(handData, ToString(card))
			}
			c.Conn.WriteJSON(map[string]interface{}{
				"event": "update_hand",
				"hand":  handData,
			})
		}

		broadcast(room, map[string]interface{}{"event": "game_started", "turn": party.Turn})
		PrintParty(party) // Log on server

		// Start the first turn
		party.StartTurn()
		PrintParty(party) // Log again to see draw
	}()
}

func handleCloseParty(client *Client) {
	roomsLock.RLock()
	room, exists := rooms[client.RoomID]
	roomsLock.RUnlock()
	if !exists {
		return
	}
	room.Mutex.Lock()
	room.Party = nil
	room.Mutex.Unlock()
	broadcast(room, map[string]string{"event": "party_closed"})
}

func broadcast(room *Room, msg interface{}) {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	for _, c := range room.Clients {
		c.Conn.WriteJSON(msg)
	}
}
