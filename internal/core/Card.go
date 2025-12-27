package core

import (
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type CardText struct {
	Description string
	Condition   string
	Effect      string
	SubEffect   *CardText
}

type RawCard struct {
	CardNumberFull string `json:"card_number_full"`
	Name           string `json:"name_face"`
	Type           string `json:"type"`
	Nation         string `json:"nation"`
	Race           string `json:"race"`
	Grade          string `json:"grade"`
	Power          string `json:"power"`
	Critical       string `json:"critical"`
	Shield         string `json:"shield"`
	Skill          string `json:"skill"`
	Gift           string `json:"gift"`
	Clan           string `json:"clan"`
	Rarity         string `json:"rarity"`
	Illustrator    string `json:"illustrator"`
	Effect         string `json:"effect"`
	Flavor         string `json:"flavor"`
}

type Boon struct {
	// TODO: Define Boon struct
}

type Card struct {
	ID             string
	CardNumberFull string
	Name           string
	Type           []string
	Nation         []string
	Race           []string
	Grade          int
	Power          int
	Critical       int
	Shield         int
	Clan           []string
	Skill          []string
	Gift           string
	Rarity         string
	Illustrator    []string
	Effect         []CardText
	Flavor         string
	Boons          []Boon
	Locked         bool
}

func ToString(card *Card) string {
	if card == nil {
		return "nil"
	}

	locked := ""
	if card.Locked {
		locked = " [LOCKED]"
	}

	return locked + "[" + card.ID + "] G" + strconv.Itoa(card.Grade) + " - " + card.Name + " => " + card.CardNumberFull + " ATK : " + strconv.Itoa(card.Power) + " DEF : " + strconv.Itoa(card.Shield) + " CRIT : " + strconv.Itoa(card.Critical)
}


func (rc *RawCard) ToCard() (*Card, error) {
	if rc == nil {
		return nil, errors.New("RawCard is nil")
	}

	UnparsedEffect := strings.Split(strings.Replace(rc.Effect, "\n・", "・", -1), "\n")
	ParsedEffects := make([]CardText, 0)

	for _, effectLine := range UnparsedEffect {
		ParsedEffects = append(ParsedEffects, CardText{
			Description: effectLine,
			// Condition:   ConditionalTrigger{}, // Types not defined yet
			// Effect:      EffectiveEffect{},    // Types not defined yet
			SubEffect: nil,
		})
	}

	grade := -1
	if rc.Grade != "" {
		grade, _ = strconv.Atoi(strings.Replace(rc.Grade, "Grade ", "", -1))
	}

	power, _ := strconv.Atoi(strings.Replace(rc.Power, "Power ", "", -1))
	critical, _ := strconv.Atoi(strings.Replace(rc.Critical, "Critical ", "", -1))
	shield, _ := strconv.Atoi(strings.Replace(rc.Shield, "Shield ", "", -1))

	return &Card{
		ID:             uuid.New().String(),
		CardNumberFull: rc.CardNumberFull,
		Name:           rc.Name,
		Type:           strings.Split(rc.Type, "/"),
		Nation:         strings.Split(rc.Nation, "/"),
		Race:           strings.Split(rc.Race, "/"),
		Grade:          grade,
		Power:          power,
		Critical:       critical,
		Shield:         shield,
		Clan:           strings.Split(rc.Clan, "/"),
		Skill:          []string{rc.Skill},
		Gift:           rc.Gift,
		Rarity:         rc.Rarity,
		Illustrator:    strings.Split(rc.Illustrator, "/"),
		Effect:         ParsedEffects,
		Flavor:         rc.Flavor,
		Boons:          []Boon{},
	}, nil
}
