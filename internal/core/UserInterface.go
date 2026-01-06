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
	Name   string
	Conn   *websocket.Conn
	RoomID string
	// Add channel to Client for Mulligan response
	MulliganCh     chan []int
	OrderCh        chan string
	RideCh         chan map[string]interface{}
	MainActionCh   chan map[string]interface{}
	BattleActionCh chan map[string]interface{}
	GuardCh        chan map[string]interface{}
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

	clientName := r.URL.Query().Get("name")
	if clientName == "" {
		clientName = "Player"
	}

	client := &Client{
		ID:             clientID,
		Name:           clientName,
		Conn:           conn,
		MulliganCh:     make(chan []int),
		OrderCh:        make(chan string),
		RideCh:         make(chan map[string]interface{}),
		MainActionCh:   make(chan map[string]interface{}),
		BattleActionCh: make(chan map[string]interface{}),
		GuardCh:        make(chan map[string]interface{}),
	}
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
			fmt.Printf("JSON Unmarshal error: %v\n", err)
			continue
		}

		action, _ := msg["action"].(string)
		fmt.Printf("Received action: %s from Client %s\n", action, clientID)
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
			handleCreateParty(client, payload)
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
		case "ride_response":
			select {
			case client.RideCh <- payload:
			default:
			}
		case "main_action_response":
			select {
			case client.MainActionCh <- payload:
			default:
			}
		case "battle_action_response":
			select {
			case client.BattleActionCh <- payload:
			default:
			}
		case "guard_response":
			select {
			case client.GuardCh <- payload:
			default:
			}
		}
	}
}

func BroadcastLog(room *Room, client *Client, message string) {
	timestamp := time.Now().Format("15:04:05")
	prefix := "System"
	if client != nil {
		prefix = fmt.Sprintf("%s#%s", client.Name, client.ID)
	}
	fullMsg := fmt.Sprintf("%s - %s : %s", timestamp, prefix, message)
	fmt.Printf("BROADCAST LOG: %s to Room %s\n", fullMsg, room.ID)
	broadcast(room, map[string]interface{}{
		"event":   "log",
		"message": fullMsg,
	})
}

func handleCreateRoom(client *Client, roomID string) {
	fmt.Printf("handleCreateRoom called for Client %s\n", client.ID)
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
	fmt.Printf("handleJoinRoom called: Client %s -> Room %s\n", client.ID, roomID)
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
	BroadcastLog(room, client, "Joined the room !")
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
			BroadcastLog(room, client, "Left the room !")
		}
	}
	client.RoomID = ""
}

func handleCreateParty(client *Client, payload map[string]interface{}) {
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

	BroadcastLog(room, client, "Created the party ! - "+client.RoomID)

	go func() {
		deck1, err1 := ParseDeckFile("decks/KT_Starter.md")
		deck2, err2 := ParseDeckFile("decks/LM_Starter.md")
		if err1 != nil || err2 != nil {
			client.Conn.WriteJSON(map[string]string{"error": "Failed to load decks"})
			return
		}

		party := InitParty([]*Deck{deck1, deck2})

		seed := ""
		if payload != nil {
			if s, ok := payload["seed"].(string); ok {
				seed = s
			}
		}

		// Force 'test1' seed for now as requested
		// if seed == "" {
		seed = "test1"
		// }
		InitGame(party, seed)

		// Set party on room
		room.Mutex.Lock()
		room.Party = party
		room.Mutex.Unlock()

		broadcast(room, map[string]interface{}{"event": "party_created"})

		BroadcastLog(room, clientsList[0], "Deck: KT_Starter")
		BroadcastLog(room, clientsList[1], "Deck: LM_Starter")
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
				BroadcastLog(room, clientsList[0], fmt.Sprintf("Rolled %d !", r0))
				BroadcastLog(room, clientsList[1], fmt.Sprintf("Rolled %d !", r1))
				time.Sleep(2 * time.Second)
			},
			func(winnerIndex int) string {
				if winnerIndex >= len(clientsList) {
					return "first"
				}
				winnerClient := clientsList[winnerIndex]
				winnerClient.Conn.WriteJSON(map[string]interface{}{"event": "ask_first_second"})

				choice := <-winnerClient.OrderCh
				BroadcastLog(room, winnerClient, fmt.Sprintf("Choose to %s", choice))
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
			BroadcastLog(room, targetClient, fmt.Sprintf("mulligan %d cards", len(indices)))
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
		// Game Loop
		for {
			party.StartTurn(func(playerIndex int, canRideFromRideDeck bool, rideDeck []*Card, hand []*Card) (string, int) {
				targetClient := clientsList[playerIndex]

				// Serialize decks for UI
				rideDeckData := []string{}
				for _, c := range rideDeck {
					rideDeckData = append(rideDeckData, c.ID) // Sending IDs, client can match or we send more data if needed.
					// For now assuming Client has card data or just needs IDs?
					// Client probably needs Names/Grades.
					// Let's send basic info or rely on Client knowing deck?
					// Let's send ToString result or similar for now to be safe/easy.
					rideDeckData = append(rideDeckData, ToString(c))
				}

				handData := []string{}
				for _, c := range hand {
					handData = append(handData, ToString(c))
				}

				vg := party.Players[playerIndex].Vanguard
				vgGrade := 0
				if vg.TopCard != nil {
					vgGrade = vg.TopCard.Grade
				}

				targetClient.Conn.WriteJSON(map[string]interface{}{
					"event":              "request_ride",
					"can_ride_from_deck": canRideFromRideDeck,
					"ride_deck":          rideDeckData,
					"hand":               handData,
					"vanguard_grade":     vgGrade,
				})

				response := <-targetClient.RideCh

				cardID := ""
				discardIdx := -1

				if val, ok := response["card_id"].(string); ok {
					cardID = val
				}
				if val, ok := response["discard_index"].(float64); ok {
					discardIdx = int(val)
				}

				BroadcastLog(room, targetClient, fmt.Sprintf("Rode %s", cardID))
				return cardID, discardIdx
			}, func(playerIndex int, hand []*Card, field map[string]*Card) (string, map[string]interface{}) {
				targetClient := clientsList[playerIndex]

				// Serialize
				handData := []string{}
				for _, c := range hand {
					handData = append(handData, ToString(c))
				}

				fieldData := make(map[string]string)
				for k, c := range field {
					if c != nil {
						fieldData[k] = ToString(c)
					} else {
						fieldData[k] = "Empty"
					}
				}

				targetClient.Conn.WriteJSON(map[string]interface{}{
					"event": "request_main_action",
					"hand":  handData,
					"field": fieldData,
				})

				response := <-targetClient.MainActionCh

				action := ""
				payload := make(map[string]interface{})

				if val, ok := response["action"].(string); ok {
					action = val
				}
				if val, ok := response["payload"].(map[string]interface{}); ok {
					payload = val
				}

				BroadcastLog(room, targetClient, fmt.Sprintf("Main Action: %s", action))
				return action, payload
			}, func(playerIndex int, field map[string]*Card) (string, map[string]interface{}) {
				targetClient := clientsList[playerIndex]

				fieldData := make(map[string]string)
				for k, c := range field {
					if c != nil {
						fieldData[k] = ToString(c)
					} else {
						fieldData[k] = "Empty"
					}
				}

				targetClient.Conn.WriteJSON(map[string]interface{}{
					"event": "request_battle_action",
					"field": fieldData,
				})

				response := <-targetClient.BattleActionCh

				action := ""
				payload := make(map[string]interface{})
				if val, ok := response["action"].(string); ok {
					action = val
				}
				if val, ok := response["payload"].(map[string]interface{}); ok {
					payload = val
				}
				BroadcastLog(room, targetClient, fmt.Sprintf("Battle Action: %s", action))
				return action, payload
			}, func(playerIndex int, hand []*Card, attackerInfo map[string]interface{}) (string, map[string]interface{}) {
				targetClient := clientsList[playerIndex]

				handData := []string{}
				for _, c := range hand {
					handData = append(handData, ToString(c))
				}

				targetClient.Conn.WriteJSON(map[string]interface{}{
					"event":         "request_guard",
					"hand":          handData,
					"attacker_info": attackerInfo,
				})

				response := <-targetClient.GuardCh

				action := ""
				payload := make(map[string]interface{})
				if val, ok := response["action"].(string); ok {
					action = val
				}
				if val, ok := response["payload"].(map[string]interface{}); ok {
					payload = val
				}
				BroadcastLog(room, targetClient, fmt.Sprintf("Guard Action: %s", action))
				return action, payload
			})
			PrintParty(party) // Log again to see draw
		}
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
	fmt.Printf("Broadcasting to %d clients in Room %s\n", len(room.Clients), room.ID)
	for _, c := range room.Clients {
		if err := c.Conn.WriteJSON(msg); err != nil {
			fmt.Printf("Broadcast write error to %s: %v\n", c.ID, err)
		}
	}
}
