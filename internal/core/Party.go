package core

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

type Deck struct {
	RideDeck [5]*Card
	MainDeck [50]*Card
	GDeck    [8]*Card
}

func findCardByNumber(cards []RawCard, targetNumber string) *RawCard {
	println("targetNumber: " + targetNumber)
	for i := range cards {
		if cards[i].CardNumberFull == targetNumber {
			// On retourne un pointeur vers l'élément trouvé
			println(cards[i].CardNumberFull)
			return &cards[i]
		}
	}
	// Si rien n'est trouvé, on retourne nil
	return nil
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
