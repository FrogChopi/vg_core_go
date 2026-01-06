package test

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const serverURL = "ws://127.0.0.1:8080/ws"

type WSMessage struct {
	Action  string                 `json:"action"`
	Payload map[string]interface{} `json:"payload"`
}

type WSEvent struct {
	Event string `json:"event"`
	// Add specific fields as needed, or access via raw map
	Hand            []string `json:"hand"`
	CanRideFromDeck bool     `json:"can_ride_from_deck"`
	RideDeck        []string `json:"ride_deck"`
	// Dice fields
	YourIndex int   `json:"your_index"`
	Rolls     []int `json:"rolls"`
}

func connectClient(t *testing.T, id string) *websocket.Conn {
	header := http.Header{}
	conn, _, err := websocket.DefaultDialer.Dial(serverURL+"?id="+id, header)
	if err != nil {
		t.Fatalf("Failed to connect client %s: %v", id, err)
	}
	t.Logf("[%s] Connected to Server", id)
	return conn
}

func sendAction(t *testing.T, conn *websocket.Conn, clientID string, action string, payload map[string]interface{}) {
	msg := WSMessage{
		Action:  action,
		Payload: payload,
	}
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("[%s] Failed to send action %s: %v", clientID, action, err)
	}
	t.Logf("[%s] Sent Action: %s | Payload: %v", clientID, action, payload)
}

func TestGameSimulation(t *testing.T) {
	// Ensure server is running before running this test!

	rand.Seed(time.Now().UnixNano())

	clientA_ID := "Client_A"
	clientB_ID := "Client_B"

	connA := connectClient(t, clientA_ID)
	defer connA.Close()

	connB := connectClient(t, clientB_ID)
	defer connB.Close()

	// 1. Create Room (Client A)
	roomID := "test_room_1"
	sendAction(t, connA, clientA_ID, "create_room", map[string]interface{}{"room_id": roomID})

	// Slight delay to ensure room creation
	time.Sleep(100 * time.Millisecond)

	// 2. Join Room (Client B)
	sendAction(t, connB, clientB_ID, "join_room", map[string]interface{}{"room_id": roomID})

	time.Sleep(100 * time.Millisecond)

	// 3. Start Game (Client A)
	sendAction(t, connA, clientA_ID, "create_party", map[string]interface{}{"seed": "test1"})

	// Event Loop
	// We listen to both clients in parallel or select?
	// For simplicity in a test, act on messages as they come for each client in goroutines.

	done := make(chan bool)

	handleClient := func(clientID string, conn *websocket.Conn) {
		role := -1 // 0 = First, 1 = Second
		myInitialIndex := -1
		mainActionCount := 0
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				t.Logf("[%s] Disconnected: %v", clientID, err)
				return
			}

			// Log Raw Message (Truncated if too long?)
			// t.Logf("[%s] Received: %s", clientID, string(message))

			var rawMap map[string]interface{}
			if err := json.Unmarshal(message, &rawMap); err != nil {
				continue
			}

			event, _ := rawMap["event"].(string)
			if event == "" {
				continue
			}

			t.Logf("[%s] Event: %s", clientID, event)

			switch event {
			case "dice_roll":
				myInitialIndex = int(rawMap["your_index"].(float64))
				t.Logf("[%s] Dice Roll: %v (You are P%d)", clientID, rawMap["rolls"], myInitialIndex)

			case "ask_first_second":
				t.Logf("[%s] Won roll, choosing First", clientID)
				sendAction(t, conn, clientID, "turn_order", map[string]interface{}{
					"order": "first",
				})

			case "turn_order":
				if msg, ok := rawMap["msg"].(string); ok {
					if strings.Contains(msg, "FIRST") {
						role = 0
					} else {
						role = 1
					}
					t.Logf("[%s] Role: Player %d", clientID, role)
				}

			case "request_mulligan":
				indices := []int{}
				if myInitialIndex == 0 { // Original P1
					indices = []int{1, 2, 4, 5}
				} else { // Original P2
					indices = []int{2, 3}
				}
				sendAction(t, conn, clientID, "mulligan_response", map[string]interface{}{
					"indices": indices,
				})
				t.Logf("[%s] Mulligan Response (Initial P%d): Discarding %v", clientID, myInitialIndex, indices)

			case "request_ride":
				// Ride from Deck, Discard 0
				rideDeckData := rawMap["ride_deck"].([]interface{})
				var cardIDToRide string
				if len(rideDeckData) > 0 {
					cardIDToRide = rideDeckData[0].(string) // Assuming format [ID, String, ID, String...]
				} else {
					cardIDToRide = "skip" // Should not happen if canRideFromDeck is true
				}

				sendAction(t, conn, clientID, "ride_response", map[string]interface{}{
					"card_id":       cardIDToRide,
					"discard_index": 0,
				})
				t.Logf("[%s] Riding from Deck %s (Discard 0)", clientID, cardIDToRide)

			case "request_main_action":
				var action string
				payload := map[string]interface{}{}

				if role == 1 && mainActionCount == 0 {
					// P1 First Action: CALL
					// Need a valid card ID from Hand
					handData := rawMap["hand"].([]interface{})
					if len(handData) > 0 {
						cardStr := handData[0].(string) // Pick first card
						start := strings.Index(cardStr, "[")
						end := strings.Index(cardStr, "]")
						if start != -1 && end != -1 {
							cardID := cardStr[start+1 : end]
							action = "CALL"
							payload = map[string]interface{}{
								"card_id":       cardID,
								"target_circle": "R1",
							}
						} else {
							action = "END"
						}
					} else {
						action = "END"
					}
				} else {
					action = "END"
				}

				sendAction(t, conn, clientID, "main_action_response", map[string]interface{}{
					"action":  action,
					"payload": payload,
				})
				t.Logf("[%s] Main Action: %s", clientID, action)
				mainActionCount++

			case "request_battle_action":
				// END
				sendAction(t, conn, clientID, "battle_action_response", map[string]interface{}{
					"action":  "END",
					"payload": map[string]interface{}{},
				})
				t.Logf("[%s] Battle Action: END", clientID)

			case "request_guard":
				// NO_GUARD
				sendAction(t, conn, clientID, "guard_response", map[string]interface{}{
					"action":  "NO_GUARD",
					"payload": map[string]interface{}{},
				})
				t.Logf("[%s] Guard Action: NO_GUARD", clientID)

			case "log":
				if msg, ok := rawMap["message"].(string); ok {
					t.Logf("[%s] GAME LOG: %s", clientID, msg)
				}

			case "error":
				t.Logf("[%s] Error from Server: %v", clientID, rawMap)
			}
		}
	}

	go handleClient(clientA_ID, connA)
	go handleClient(clientB_ID, connB)

	// Run for a fixed time or until condition?
	// Let's run for 10 seconds to cover a few turns.
	time.Sleep(10 * time.Second)
	close(done)
}
