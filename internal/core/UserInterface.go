package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

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

	client := &Client{ID: clientID, Conn: conn}
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
			handleCreateRoom(client)
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
		}
	}
}

func handleCreateRoom(client *Client) {
	handleQuitRoom(client)

	roomID := uuid.New().String()
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
	defer room.Mutex.Unlock()

	if len(room.Clients) < 2 {
		client.Conn.WriteJSON(map[string]string{"error": "Need at least 2 players"})
		return
	}

	deck1, err1 := ParseDeckFile("decks/KT_Starter.md")
	deck2, err2 := ParseDeckFile("decks/LM_Starter.md")
	if err1 != nil || err2 != nil {
		client.Conn.WriteJSON(map[string]string{"error": "Failed to load decks"})
		return
	}

	party := InitParty([]*Deck{deck1, deck2})
	InitGame(party, "")
	room.Party = party

	broadcast(room, map[string]interface{}{"event": "party_created", "turn": party.Turn})
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
	for _, c := range room.Clients {
		c.Conn.WriteJSON(msg)
	}
}