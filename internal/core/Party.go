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
	FuncCall  func() `json:"-"`
}

type Deck struct {
	RideDeck [5]*Card
	MainDeck [50]*Card
	GDeck    [8]*Card
}

type Circle struct {
	TopCard   *Card
	Soul	  *Card
	Boon      *Card
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
	Rear1  		Circle
	Vanguard  	Circle
	Rear2  		Circle
	Rear3  		Circle
	Rear4  		Circle
	Rear5  		Circle
}

type Party struct {
	seed    	string
	rand        *rand.Rand
	Players     []Player
	Turn    	int
	EventQueue  []Event
	History     []Event
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
		Rear1:     Circle{},
		Vanguard:  Circle{},
		Rear2:     Circle{},
		Rear3:     Circle{},
		Rear4:     Circle{},
		Rear5:     Circle{},
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