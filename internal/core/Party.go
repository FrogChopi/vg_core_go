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
	Soul    *Card
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
func (party *Party) StartTurn() {
	party.Turn++

	player := &party.Players[(party.Turn-1)%len(party.Players)]
	println("Turn", party.Turn, "starts for Player", (party.Turn-1)%len(party.Players))

	party.StandPhase(player)
	party.DrawPhase(player)
	party.RidePhase(player)
	party.MainPhase(player)
	party.BattlePhase(player)
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

func (party *Party) RidePhase(player *Player) {
	party.ProcessPhase(PhaseRide, func() {
		// Logic to wait for user input for Ride would go here
	})
}

func (party *Party) MainPhase(player *Player) {
	party.ProcessPhase(PhaseMain, func() {
		// Main phase actions
	})
}

func (party *Party) BattlePhase(player *Player) {
	party.ProcessPhase(PhaseBattle, func() {
		// Battle phase actions
	})
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
