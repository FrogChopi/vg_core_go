package core

import (
	"bufio"
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type Event struct {
	EventType string
	Origin    string
	FuncCall  func()
}

type Deck struct {
	RideDeck [5]*Card
	MainDeck [50]*Card
	GDeck    [8]*Card
}

type Circle struct {
	TopCard *Card
	Soul    []*Card
	Boon    *Card
}

type Player struct {
	RideDeck    []*Card
	MainDeck    []*Card
	GDeck       []*Card
	DamageZone  []*Card
	Hand        []*Card
	OrderZone   []*Card
	GuardZone   []*Card
	TriggerZone []*Card
	BindZone    []*Card
	DropZone    []*Card
	Rear1       Circle
	Vanguard    Circle
	Rear2       Circle
	Rear3       Circle
	Rear4       Circle
	Rear5       Circle

	// Turn State
	OrderPlayedThisTurn bool
}

// BattleState tracks the current attack state
type BattleState struct {
	Attacker       *Card
	Defender       *Card
	AttackerCircle *Circle
	DefenderCircle *Circle

	CurrentPower int
	GuardPower   int
	CritCount    int
	DriveCount   int

	IsBoosted bool
	Booster   *Card
}

const (
	PhaseStand  = "Stand Phase"
	PhaseDraw   = "Draw Phase"
	PhaseRide   = "Ride Phase"
	PhaseMain   = "Main Phase"
	PhaseBattle = "Battle Phase"
	PhaseEnd    = "End Phase"
)

type Party struct {
	seed         string
	rand         *rand.Rand
	Players      []Player
	Turn         int
	CurrentPhase string
	EventQueue   []Event
	History      []Event
}

func (party *Party) checkEffects(trigger string) {
	// Placeholder: In the future, iterate through cards/effects to see if any trigger matches
	// fmt.Println("Checking effects for:", trigger)
}

// ProcessPhase executes the standard flow of a phase: Start Effects -> Action -> End Effects
func (party *Party) ProcessPhase(phaseName string, defaultAction func()) {
	party.CurrentPhase = phaseName
	// println("Processing " + phaseName)

	// 1. Start of Phase Effects
	party.checkEffects("START_" + strings.ToUpper(strings.ReplaceAll(phaseName, " ", "_")))

	// 2. Action
	// In a full implementation, we would check if an effect REPLACES the default action here.
	if defaultAction != nil {
		defaultAction()
	}

	// 3. End of Phase Effects
	party.checkEffects("END_" + strings.ToUpper(strings.ReplaceAll(phaseName, " ", "_")))
}

// StartTurn executes the phases for the current turn's player
func (party *Party) StartTurn(
	requestRide func(int, bool, []*Card, []*Card) (string, int),
	requestMainAction func(int, []*Card, map[string]*Card) (string, map[string]interface{}),
	requestBattleAction func(int, map[string]*Card) (string, map[string]interface{}),
	requestGuard func(int, []*Card, map[string]interface{}) (string, map[string]interface{}),
) {
	party.Turn++

	player := &party.Players[(party.Turn-1)%len(party.Players)]
	println("Turn", party.Turn, "starts for Player", (party.Turn-1)%len(party.Players))

	// Reset Turn State
	player.OrderPlayedThisTurn = false

	party.StandPhase(player)
	party.DrawPhase(player)
	party.RidePhase(player, requestRide)
	party.MainPhase(player, requestMainAction)
	party.BattlePhase(player, requestBattleAction, requestGuard)
	party.EndPhase(player)
}

func (party *Party) StandPhase(player *Player) {
	party.ProcessPhase(PhaseStand, func() {
		// Stand all units
		// Placeholder: iterate over circles and set IsStand = true (need to add IsStand to Card or Circle)
		// For now, just logging
		// println("Standing units...")
	})
}

func (party *Party) DrawPhase(player *Player) {
	party.ProcessPhase(PhaseDraw, func() {
		// Standard Draw: Draw 1 card
		// Effects could replace this (e.g., skip draw to do X) if implemented in checkEffects/middleware
		draw(player, 1)
	})
}

func (party *Party) RidePhase(player *Player, requestRide func(playerIndex int, canRideFromRideDeck bool, rideDeck []*Card, hand []*Card) (string, int)) {
	party.ProcessPhase(PhaseRide, func() {
		// 1. Identify Valid Ride Targets
		currentGrade := 0
		if player.Vanguard.TopCard != nil {
			currentGrade = player.Vanguard.TopCard.Grade
		}

		canRideFromRideDeck := false
		// Check Ride Deck (simplification: just check if next grade exists)
		// Standard: Ride Deck contains 0, 1, 2, 3.
		// If current is 0, we look for 1.
		for _, c := range player.RideDeck {
			if c.Grade == currentGrade+1 {
				canRideFromRideDeck = true
				break
			}
		}

		// 2. Request Input
		// We pass the full ride deck and hand for the UI to display options
		playerIdx := -1
		for i := range party.Players {
			if &party.Players[i] == player {
				playerIdx = i
				break
			}
		}

		cardIDToRide, discardIdx := requestRide(playerIdx, canRideFromRideDeck, player.RideDeck, player.Hand)

		if cardIDToRide == "skip" {
			return
		}

		// 3. Process Ride
		var cardToRide *Card
		isFromRideDeck := false
		rideDeckIdx := -1

		// Search in Ride Deck
		for i, c := range player.RideDeck {
			if c.ID == cardIDToRide {
				cardToRide = c
				isFromRideDeck = true
				rideDeckIdx = i
				break
			}
		}

		// Search in Hand
		if cardToRide == nil {
			for _, c := range player.Hand {
				if c.ID == cardIDToRide {
					cardToRide = c
					break
				}
			}
		}

		if cardToRide == nil {
			return // Invalid ID
		}

		// Validate Grade
		// user rule: "on ne peut ride que un niveau égal ou supérieur de 1 au vanguard actuel"
		// implementation: Allow Grade == currentGrade OR Grade == currentGrade + 1
		if cardToRide.Grade != currentGrade && cardToRide.Grade != currentGrade+1 {
			return // Invalid Grade
		}

		// Cost (if from Ride Deck)
		if isFromRideDeck {
			// Unless effect says otherwise (omitted)
			if discardIdx >= 0 && discardIdx < len(player.Hand) {
				// Discard
				discarded := player.Hand[discardIdx]
				player.DropZone = append(player.DropZone, discarded)

				// Remove from hand
				// Efficient removal
				player.Hand = append(player.Hand[:discardIdx], player.Hand[discardIdx+1:]...)
			} else {
				// Determine if cost is mandatory? Yes usually.
				// If no valid discard provided, cancel ride?
				// For simplicity assuming valid input or providing cancel logic in UI.
				// If fail to discard, abort.
				return
			}

			// Remove from Ride Deck
			player.RideDeck = append(player.RideDeck[:rideDeckIdx], player.RideDeck[rideDeckIdx+1:]...)
		} else {
			// Remove from Hand
			// Need to find index again or use discard logic
			newHand := []*Card{}
			for _, c := range player.Hand {
				if c.ID != cardIDToRide {
					newHand = append(newHand, c)
				}
			}
			player.Hand = newHand
		}

		// 4. Ride!
		if player.Vanguard.TopCard != nil {
			player.Vanguard.Soul = append(player.Vanguard.Soul, player.Vanguard.TopCard)
		}
		player.Vanguard.TopCard = cardToRide

		// 5. Effects
		party.checkEffects("ON_RIDE")

	})
}

func (party *Party) MainPhase(player *Player, requestMainAction func(playerIndex int, hand []*Card, field map[string]*Card) (string, map[string]interface{})) {
	party.ProcessPhase(PhaseMain, func() {
		// Main phase actions loop
		for {
			playerIdx := -1
			for i := range party.Players {
				if &party.Players[i] == player {
					playerIdx = i
					break
				}
			}

			// Construct simplified field map for UI
			field := make(map[string]*Card)
			field["R1"] = player.Rear1.TopCard
			field["R2"] = player.Rear2.TopCard
			field["R3"] = player.Rear3.TopCard
			field["R4"] = player.Rear4.TopCard
			field["R5"] = player.Rear5.TopCard
			field["V"] = player.Vanguard.TopCard

			action, payload := requestMainAction(playerIdx, player.Hand, field)

			if action == "END" {
				break
			}

			if action == "CALL" {
				// Payload: { "card_id": string, "target_circle": string (R1-R5) }
				cardID, _ := payload["card_id"].(string)
				targetCircleName, _ := payload["target_circle"].(string)

				// Find Card in Hand
				var cardToCall *Card
				handIdx := -1
				for i, c := range player.Hand {
					if c.ID == cardID {
						cardToCall = c
						handIdx = i
						break
					}
				}

				if cardToCall == nil {
					continue
				}

				// Grade Check
				vGrade := 0
				if player.Vanguard.TopCard != nil {
					vGrade = player.Vanguard.TopCard.Grade
				}
				if cardToCall.Grade > vGrade {
					continue
				}

				// Determine Circle
				var circle *Circle
				switch targetCircleName {
				case "R1":
					circle = &player.Rear1
				case "R2":
					circle = &player.Rear2
				case "R3":
					circle = &player.Rear3
				case "R4":
					circle = &player.Rear4
				case "R5":
					circle = &player.Rear5
				}

				if circle != nil {
					// Retire existing
					if circle.TopCard != nil {
						player.DropZone = append(player.DropZone, circle.TopCard)
					}
					// Place card
					circle.TopCard = cardToCall

					// Remove from Hand
					player.Hand = append(player.Hand[:handIdx], player.Hand[handIdx+1:]...)

					party.checkEffects("ON_CALL")
				}
			} else if action == "MOVE" {
				// Payload: { "circle_1": string, "circle_2": string }
				c1Name, _ := payload["circle_1"].(string)
				c2Name, _ := payload["circle_2"].(string)

				// Validate Swap (R1<->R3 or R2<->R5)
				valid := false
				if (c1Name == "R1" && c2Name == "R3") || (c1Name == "R3" && c2Name == "R1") {
					valid = true
				}
				if (c1Name == "R2" && c2Name == "R5") || (c1Name == "R5" && c2Name == "R2") {
					valid = true
				}

				if valid {
					// Get Circles
					// Helper to get circle ptr by name?
					var circ1, circ2 *Circle

					getCircle := func(name string) *Circle {
						switch name {
						case "R1":
							return &player.Rear1
						case "R2":
							return &player.Rear2
						case "R3":
							return &player.Rear3
						case "R4":
							return &player.Rear4
						case "R5":
							return &player.Rear5
						}
						return nil
					}
					circ1 = getCircle(c1Name)
					circ2 = getCircle(c2Name)

					if circ1 != nil && circ2 != nil {
						circ1.TopCard, circ2.TopCard = circ2.TopCard, circ1.TopCard
					}
				}

			} else if action == "PLAY_ORDER" {
				// Payload: { "card_id": string }
				if player.OrderPlayedThisTurn {
					continue
				}

				cardID, _ := payload["card_id"].(string)
				var cardToPlay *Card
				handIdx := -1
				for i, c := range player.Hand {
					if c.ID == cardID {
						cardToPlay = c
						handIdx = i
						break
					}
				}

				if cardToPlay == nil {
					continue
				}

				// Grade check
				vGrade := 0
				if player.Vanguard.TopCard != nil {
					vGrade = player.Vanguard.TopCard.Grade
				}
				if cardToPlay.Grade > vGrade {
					continue
				}

				// Determine Type (Set Order vs Normal)
				// Assuming Type is string slice.
				isOrder := false
				isSet := false
				for _, t := range cardToPlay.Type {
					if strings.Contains(t, "Order") {
						isOrder = true
					}
					if strings.Contains(t, "Set") {
						isSet = true
					}
				}

				if !isOrder {
					continue
				}

				player.OrderPlayedThisTurn = true

				// Remove from Hand
				player.Hand = append(player.Hand[:handIdx], player.Hand[handIdx+1:]...)

				if isSet {
					// Put in Order Zone
					player.OrderZone = append(player.OrderZone, cardToPlay)
				} else {
					// Normal Order -> Effect -> Drop
					// Put in Drop
					player.DropZone = append(player.DropZone, cardToPlay)
				}
			} else if action == "ACT" {
				// Placeholder
			}
		}
	})
}

func (party *Party) BattlePhase(player *Player,
	requestBattleAction func(playerIndex int, field map[string]*Card) (string, map[string]interface{}),
	requestGuard func(playerIndex int, hand []*Card, attackerInfo map[string]interface{}) (string, map[string]interface{}),
) {
	party.ProcessPhase(PhaseBattle, func() {
		// Rule: First Player cannot attack on Turn 1
		if party.Turn == 1 {
			println("Turn 1 - Battle Phase Skipped (Cannot Attack)")
			return
		}
		// Battle phase actions loop
		for {
			// 1. Request Action
			playerIdx := -1
			for i := range party.Players {
				if &party.Players[i] == player {
					playerIdx = i
					break
				}
			}

			field := make(map[string]*Card)
			field["R1"] = player.Rear1.TopCard
			field["R2"] = player.Rear2.TopCard
			field["R3"] = player.Rear3.TopCard
			field["R4"] = player.Rear4.TopCard
			field["R5"] = player.Rear5.TopCard
			field["V"] = player.Vanguard.TopCard

			action, payload := requestBattleAction(playerIdx, field)

			if action == "END" {
				break
			}

			if action == "ATTACK" {
				// Payload: { "attacker": "V"/"R1"..., "target": "V" ... }
				party.ResolveBattle(player, payload, requestGuard)
			}
		}
	})
}

// BattleState struct (assuming it's defined elsewhere, e.g., in types.go)
// type BattleState struct {
// 	Attacker       *Card
// 	Defender       *Card
// 	AttackerCircle *Circle
// 	DefenderCircle *Circle
// 	Booster        *Card
// 	CurrentPower   int
// 	CritCount      int
// 	DriveCount     int
// 	GuardPower     int
// 	IsBoosted      bool
// }

func (party *Party) ResolveBattle(player *Player, payload map[string]interface{}, requestGuard func(playerIndex int, hand []*Card, attackerInfo map[string]interface{}) (string, map[string]interface{})) {
	attackerCircleStr, _ := payload["attacker"].(string)
	targetCircleStr, _ := payload["target"].(string)

	var attackerCircle *Circle
	var defenderCircle *Circle

	// Find Attacker Circle
	switch attackerCircleStr {
	case "V":
		attackerCircle = &player.Vanguard
	case "R1":
		attackerCircle = &player.Rear1
	case "R2":
		attackerCircle = &player.Rear2
	case "R3":
		attackerCircle = &player.Rear3
	case "R4":
		attackerCircle = &player.Rear4
	case "R5":
		attackerCircle = &player.Rear5
	}

	// Find Opponent
	playerIdx := -1
	for i := range party.Players {
		if &party.Players[i] == player {
			playerIdx = i
			break
		}
	}
	opponentIdx := (playerIdx + 1) % len(party.Players) // Assuming 2 players
	opponent := &party.Players[opponentIdx]

	// Find Defender Circle
	switch targetCircleStr {
	case "V":
		defenderCircle = &opponent.Vanguard
	case "R1":
		defenderCircle = &opponent.Rear1
	case "R2":
		defenderCircle = &opponent.Rear2
	case "R3":
		defenderCircle = &opponent.Rear3
	case "R4":
		defenderCircle = &opponent.Rear4
	case "R5":
		defenderCircle = &opponent.Rear5
	}

	if attackerCircle == nil || defenderCircle == nil || attackerCircle.TopCard == nil || defenderCircle.TopCard == nil {
		println("Invalid Battle: Missing Unit")
		return
	}

	state := &BattleState{
		Attacker:       attackerCircle.TopCard,
		Defender:       defenderCircle.TopCard,
		AttackerCircle: attackerCircle,
		DefenderCircle: defenderCircle,
		CurrentPower:   attackerCircle.TopCard.Power,
		CritCount:      attackerCircle.TopCard.Critical,
		DriveCount:     0, // Will be set during drive check
		GuardPower:     0,
	}

	println("Battle Start:", state.Attacker.Name, "attacks", state.Defender.Name)

	// Boost Step
	// Check card behind attacker
	var boosterCircle *Circle
	if attackerCircle == &player.Rear1 {
		boosterCircle = &player.Rear3
	} else if attackerCircle == &player.Rear2 {
		boosterCircle = &player.Rear5
	} else if attackerCircle == &player.Vanguard {
		boosterCircle = &player.Rear4
	}

	if boosterCircle != nil && boosterCircle.TopCard != nil {
		// Check for Boost skill
		canBoost := false
		for _, s := range boosterCircle.TopCard.Skill {
			if s == "Boost" {
				canBoost = true
				break
			}
		}
		if canBoost {
			state.IsBoosted = true
			state.Booster = boosterCircle.TopCard
			state.CurrentPower += boosterCircle.TopCard.Power
			println("Boosted by", state.Booster.Name, "+", state.Booster.Power)
		}
	}

	// Guard Step
	for {
		attackerInfo := map[string]interface{}{
			"power":    state.CurrentPower,
			"shield":   state.GuardPower,
			"crit":     state.CritCount,
			"attacker": state.Attacker.Name,
		}

		action, gPayload := requestGuard(opponentIdx, opponent.Hand, attackerInfo)

		if action == "NO_GUARD" || action == "END" {
			break
		}

		if action == "GUARD" {
			// Card ID in payload
			cardID, _ := gPayload["card_id"].(string)
			// Check Hand
			handIdx := -1
			var card *Card
			for i, c := range opponent.Hand {
				if c.ID == cardID {
					handIdx = i
					card = c
					break
				}
			}

			if card != nil {
				// Move from Hand to Guard Zone (Drop for now or temp zone?)
				// Usually goes to GC then Drop. Simplification: Add Shield, Move to Drop.
				state.GuardPower += card.Shield
				println("Guarded with", card.Name, "Shield:", card.Shield)
				opponent.Hand = append(opponent.Hand[:handIdx], opponent.Hand[handIdx+1:]...)
				opponent.DropZone = append(opponent.DropZone, card)
			}
		}
	}

	// Drive Check
	if state.AttackerCircle == &player.Vanguard {
		// Determine Drive Count
		// Assume Twin Drive (2 checks) if not specified or check skill.
		checkCount := 1 // Default
		for _, s := range state.Attacker.Skill {
			if s == "Twin Drive!!" {
				checkCount = 2
				break
			} else if s == "Triple Drive!!!" {
				checkCount = 3
				break
			}
		}

		party.PerformDriveChecks(player, checkCount, state)
	}

	// Result
	totalAttack := state.CurrentPower
	totalDef := state.Defender.Power + state.GuardPower

	println("Battle Result: Atk", totalAttack, "vs Def", totalDef)

	if totalAttack >= totalDef {
		println("Hit!")
		// Hit Logic
		if state.DefenderCircle == &opponent.Vanguard {
			party.PerformDamageChecks(opponent, state.CritCount)
		} else {
			// Retire Rear-Guard
			println("Rear-Guard Retired")
			// Move to Drop
			opponent.DropZone = append(opponent.DropZone, state.Defender)
			state.DefenderCircle.TopCard = nil
		}
	} else {
		println("Attack Guarded/Missed")
	}
}

func (party *Party) PerformDriveChecks(player *Player, count int, state *BattleState) {
	println("Perform Drive Check:", count)
	for i := 0; i < count; i++ {
		if len(player.MainDeck) == 0 {
			println("Deck Empty!")
			return
		}
		card := player.MainDeck[0]
		player.MainDeck = player.MainDeck[1:]

		// Put in Trigger Zone (Field field?) or just temp
		println("Drive Check:", card.Name)

		// Trigger Logic
		// Check triggers
		isTrigger := false

		for _, t := range card.Type {
			if strings.Contains(t, "Trigger") {
				isTrigger = true
				// Apply effects
				// Simplified: all triggers give +10000 power to one unit (Attacker here)
				state.CurrentPower += 10000 // Power to attacking unit
				println("Trigger! +10000 Power")

				if strings.Contains(t, "Critical Trigger") {
					state.CritCount += 1 // Crit to attacking unit
					println("Critical Trigger! +1 Crit")
				} else if strings.Contains(t, "Front Trigger") {
					// All front row units get +10000 power
					if player.Vanguard.TopCard != nil {
						player.Vanguard.TopCard.Power += 10000
					}
					if player.Rear1.TopCard != nil {
						player.Rear1.TopCard.Power += 10000
					}
					if player.Rear2.TopCard != nil {
						player.Rear2.TopCard.Power += 10000
					}
					println("Front Trigger! Front Row +10000")
				} else if strings.Contains(t, "Heal Trigger") {
					// Heal 1 damage if damage is >= opponent's damage
					// Simplified: just heal 1 damage if possible
					if len(player.DamageZone) > 0 {
						healedCard := player.DamageZone[len(player.DamageZone)-1]
						player.DamageZone = player.DamageZone[:len(player.DamageZone)-1]
						player.DropZone = append(player.DropZone, healedCard)
						println("Healed 1 damage!")
					}
				} else if strings.Contains(t, "Draw Trigger") {
					// Draw 1 card
					if len(player.MainDeck) > 0 {
						drawnCard := player.MainDeck[0]
						player.MainDeck = player.MainDeck[1:]
						player.Hand = append(player.Hand, drawnCard)
						println("Drew 1 card!")
					}
				} else if strings.Contains(t, "Over Trigger") {
					// Remove from game, draw 1, +100 Million?
					// Simplified: +100 Million Power
					state.CurrentPower += 100000000
					println("Over Trigger! +100 Million Power")
				}
				break // Only one trigger type per card
			}
		}

		if isTrigger {
			// Usually triggers go to hand except Over Trigger?
			// Standard: Drive Checks go to hand.
		}

		// Add to Hand
		player.Hand = append(player.Hand, card)
	}
}

func (party *Party) PerformDamageChecks(player *Player, count int) {
	println("Perform Damage Check:", count)
	for i := 0; i < count; i++ {
		if len(player.MainDeck) == 0 {
			println("Deck Empty! No more damage to take.")
			return
		}
		card := player.MainDeck[0]
		player.MainDeck = player.MainDeck[1:]

		println("Damage Check:", card.Name)

		// Trigger Logic
		for _, t := range card.Type {
			if strings.Contains(t, "Trigger") {
				println("Damage Trigger! +10000 Power to Vanguard")
				// Power +10k to Vanguard
				if player.Vanguard.TopCard != nil {
					player.Vanguard.TopCard.Power += 10000
				}

				// Effects
				if strings.Contains(t, "Draw Trigger") {
					// Draw
					if len(player.MainDeck) > 0 {
						drawnCard := player.MainDeck[0]
						player.MainDeck = player.MainDeck[1:]
						player.Hand = append(player.Hand, drawnCard)
						println("Drew 1 card!")
					}
				} else if strings.Contains(t, "Heal Trigger") {
					// Heal? Logic complex in damage check (if you heal the card you just got? or existing?)
					// Rules: "Put top card into trigger zone. Resolve effects. Put into damage zone."
					// So you heal an existing card, then put this one in.
					if len(player.DamageZone) > 0 {
						// Find card to heal? Random/Last.
						healedCard := player.DamageZone[len(player.DamageZone)-1]
						player.DamageZone = player.DamageZone[:len(player.DamageZone)-1]
						player.DropZone = append(player.DropZone, healedCard)
						println("Healed 1 damage!")
					}
				}
				// Crit and Front triggers apply effects too but often less relevant on defense,
				// except power.
				break
			}
		}

		player.DamageZone = append(player.DamageZone, card)

		if len(player.DamageZone) >= 6 {
			println("Game Over! Player Lost.")
		}
	}
}

func (party *Party) EndPhase(player *Player) {
	party.ProcessPhase(PhaseEnd, func() {
		// End of turn effects
	})
}

func PrintDeck(deck *Deck) {
	println("Ride Deck: [")
	for _, card := range deck.RideDeck {
		println("\t" + ToString(card))
	}
	println("]\n")
	println("Main Deck: [")
	for _, card := range deck.MainDeck {
		println("\t" + ToString(card))
	}
	println("]\n")
	println("G Deck: [")
	for _, card := range deck.GDeck {
		println("\t" + ToString(card))
	}
	println("]")
}

func PrintParty(party *Party) {
	println("Turn: " + strconv.Itoa(party.Turn))
	for i, player := range party.Players {

		println("\n====================")
		println("Player " + strconv.Itoa(i) + ":")
		println("====================")

		println("Hand: [")
		for _, card := range player.Hand {
			println("\t" + ToString(card))
		}
		println("]\n")

		println("====================")

		print("R1 : ")
		if player.Rear1.TopCard != nil {
			println("\t" + ToString(player.Rear1.TopCard))
		}

		print("V : ")
		if player.Vanguard.TopCard != nil {
			println("\t" + ToString(player.Vanguard.TopCard))
		}

		print("R2 : ")
		if player.Rear2.TopCard != nil {
			println("\t" + ToString(player.Rear2.TopCard))
		}

		print("R3 : ")
		if player.Rear3.TopCard != nil {
			println("\t" + ToString(player.Rear3.TopCard))
		}

		print("R4 : ")
		if player.Rear4.TopCard != nil {
			println("\t" + ToString(player.Rear4.TopCard))
		}

		print("R5 : ")
		if player.Rear5.TopCard != nil {
			println("\t" + ToString(player.Rear5.TopCard))
		}

		println("\n====================")

		println("Damage Zone: [")
		for _, card := range player.DamageZone {
			println("\t" + ToString(card))
		}
		println("]\n")

		println("====================")

		println("Drop Zone: [")
		for _, card := range player.DropZone {
			println("\t" + ToString(card))
		}
		println("]\n")

		println("====================")

		println("Bind Zone: [")
		for _, card := range player.BindZone {
			println("\t" + ToString(card))
		}
		println("]\n")

	}
}

func findCardByNumber(cards []RawCard, targetNumber string) *RawCard {
	// println("targetNumber: " + targetNumber)
	for i := range cards {
		if cards[i].CardNumberFull == targetNumber {
			// On retourne un pointeur vers l'élément trouvé
			// println(cards[i].CardNumberFull)
			return &cards[i]
		}
	}
	// Si rien n'est trouvé, on retourne nil
	return nil
}

func DeckToPlayer(deck Deck) Player {
	return Player{
		RideDeck:    deck.RideDeck[:],
		MainDeck:    deck.MainDeck[:],
		GDeck:       deck.GDeck[:],
		DamageZone:  []*Card{},
		Hand:        []*Card{},
		OrderZone:   []*Card{},
		GuardZone:   []*Card{},
		TriggerZone: []*Card{},
		BindZone:    []*Card{},
		DropZone:    []*Card{},
		Rear1:       Circle{},
		Vanguard:    Circle{},
		Rear2:       Circle{},
		Rear3:       Circle{},
		Rear4:       Circle{},
		Rear5:       Circle{},
	}
}

func ParseDeckFile(filePath string) (*Deck, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	RideDeck := [][]string{}
	MainDeck := [][]string{}
	GDeck := [][]string{}

	scanner := bufio.NewScanner(file)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignorer les lignes vides
		if line == "" {
			continue
		}

		// Détection de la section
		if strings.HasPrefix(line, "#") {
			currentSection = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "#")))
			continue
		}

		// Parsing de la ligne de carte
		cardData := parseCardLine(line)

		// Ajout à la bonne section
		switch currentSection {
		case "ride":
			RideDeck = append(RideDeck, cardData)
		case "main":
			MainDeck = append(MainDeck, cardData)
		case "g":
			GDeck = append(GDeck, cardData)
		}
	}

	databaseFile, err := os.Open("vg_parsed_cards.json")
	if err != nil {
		return nil, err
	}
	defer databaseFile.Close()

	var database []RawCard
	if err := json.NewDecoder(databaseFile).Decode(&database); err != nil {
		return nil, err
	}

	deck := &Deck{}

	// Helper function to process deck sections
	processSection := func(sectionData [][]string, targetArray []*Card) ([]*Card, error) {
		result := []*Card{}
		for _, cardData := range sectionData {
			// Remove 'x' from count (e.g., "4x" -> "4")
			countStr := strings.TrimSuffix(cardData[0], "x")
			count, err := strconv.Atoi(countStr)
			if err != nil {
				return nil, err
			}
			cardNumber := cardData[3]

			rawCard := findCardByNumber(database, cardNumber)
			var card *Card
			if rawCard != nil {
				card, err = rawCard.ToCard()
				if err != nil {
					return nil, err
				}
			} else {
				// Keep invalid/not found cards as nil
				card = nil
			}

			for i := 0; i < count; i++ {
				result = append(result, card)
			}
		}
		return result, nil
	}

	// Process Ride Deck
	rideCards, err := processSection(RideDeck, deck.RideDeck[:0])
	if err != nil {
		return nil, err
	}
	copy(deck.RideDeck[:], rideCards)

	// Process Main Deck
	mainCards, err := processSection(MainDeck, deck.MainDeck[:0])
	if err != nil {
		return nil, err
	}
	copy(deck.MainDeck[:], mainCards)

	// Process G Deck
	gCards, err := processSection(GDeck, deck.GDeck[:0])
	if err != nil {
		return nil, err
	}
	copy(deck.GDeck[:], gCards)

	return deck, nil
}

// parseCardLine découpe la ligne intelligemment
func parseCardLine(line string) []string {
	// 1. Extraire la quantité (ex: 1x)
	return strings.Split(line, "\t")
}

func InitParty(decks []*Deck) *Party {
	var players []Player

	for _, deck := range decks {
		if deck != nil {
			players = append(players, DeckToPlayer(*deck))
		}
	}

	return &Party{
		Players:    players,
		Turn:       0,
		EventQueue: []Event{},
		History:    []Event{},
	}
}

func draw(player *Player, count int) bool {
	if len(player.MainDeck) >= count {
		for i := 0; i < count; i++ {
			player.Hand = append(player.Hand, player.MainDeck[0])
			player.MainDeck = player.MainDeck[1:]
		}
		return true
	}
	return false
}

func InitGame(party *Party, seed string) {

	if seed == "" {
		seed = strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	party.seed = seed
	seedInt, _ := strconv.ParseInt(seed, 10, 64)
	party.rand = rand.New(rand.NewSource(seedInt))

	// Initial game setup logic here
	for i := range party.Players {
		player := &party.Players[i]

		party.rand.Shuffle(len(player.MainDeck), func(i, j int) {
			player.MainDeck[i], player.MainDeck[j] = player.MainDeck[j], player.MainDeck[i]
		})

		for j, card := range player.RideDeck {
			if card != nil && card.Grade == 0 {
				player.Vanguard.TopCard = card
				println("Vanguard : " + ToString(card))
				card.Locked = true
				player.RideDeck = append(player.RideDeck[:j], player.RideDeck[j+1:]...)
				break
			}
		}

		draw(player, 5)
	}
}

// DecideTurnOrder simulation
// Returns true if the players were swapped (i.e. original P1 becomes P0)
func (party *Party) DecideTurnOrder(onRoll func(int, int), askChoice func(winnerIndex int) string) bool {
	for {
		r0 := party.rand.Intn(6) + 1
		r1 := party.rand.Intn(6) + 1

		onRoll(r0, r1)

		if r0 != r1 {
			winner := 0
			if r1 > r0 {
				winner = 1
			}
			choice := askChoice(winner)
			// If winner chooses second, swap
			// Default winner is P0 (index winner)
			// If P0 wins and chooses Second -> Swap
			// If P1 wins and chooses First -> Swap (so P1 becomes P0)

			doSwap := false
			if winner == 0 && choice == "second" {
				doSwap = true
			} else if winner == 1 && choice == "first" {
				doSwap = true
			}

			if doSwap {
				party.Players[0], party.Players[1] = party.Players[1], party.Players[0]
			}
			return doSwap
		}
		time.Sleep(1 * time.Second)
	}
}

// PerformMulligan executes the Mulligan phase in PARALLEL.
func (party *Party) PerformMulligan(requestMulligan func(playerIndex int, hand []*Card) []int) {
	type result struct {
		Index   int
		Indices []int
	}
	results := make(chan result, len(party.Players))

	// 1. Request mulligans in parallel
	for i := range party.Players {
		go func(idx int) {
			// Provide a copy of hand or just reference, requestMulligan uses it for display
			// Accessing party.Players[idx].Hand is safe for reading here as main thread waits
			res := requestMulligan(idx, party.Players[idx].Hand)
			results <- result{Index: idx, Indices: res}
		}(i)
	}

	// 2. Gather results
	discardMap := make(map[int][]int)
	for i := 0; i < len(party.Players); i++ {
		res := <-results
		discardMap[res.Index] = res.Indices
	}

	// 3. Apply changes
	for i := range party.Players {
		player := &party.Players[i]
		discardIndices := discardMap[i]

		validIndices := []int{}
		for _, idx := range discardIndices {
			if idx >= 0 && idx < len(player.Hand) {
				validIndices = append(validIndices, idx)
			}
		}

		if len(validIndices) > 0 {
			cardsToDiscard := []*Card{}
			newHand := []*Card{}
			tempDeck := []*Card{}

			isDiscarded := make(map[int]bool)
			for _, idx := range validIndices {
				isDiscarded[idx] = true
			}

			for hIdx, card := range player.Hand {
				if isDiscarded[hIdx] {
					tempDeck = append(tempDeck, card)
					cardsToDiscard = append(cardsToDiscard, card)
				} else {
					newHand = append(newHand, card)
				}
			}

			player.Hand = newHand
			player.MainDeck = append(player.MainDeck, tempDeck...)

			party.rand.Shuffle(len(player.MainDeck), func(i, j int) {
				player.MainDeck[i], player.MainDeck[j] = player.MainDeck[j], player.MainDeck[i]
			})

			draw(player, len(cardsToDiscard))
		}
	}
}
