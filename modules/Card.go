import (
	"fmt"
	"github.com/google/uuid"
)

type CardText struct {
	Description string
	Condition ConditionalTrigger
	Effect EffectiveEffect
	SubEffect *CardText
}

type RawCard struct {
	CardNumberFull string `json:"card_number_full"`
	Name string `json:"name_face"`
	Type string `json:"type"`
	Nation string `json:"nation"`
	Race string `json:"race"`
	Grade string `json:"grade"`
	Power string `json:"power"`
	Critical string `json:"critical"`
	Shield string `json:"shield"`
	Skill string `json:"skill"`
	Gift string `json:"gift"`
	Clan string `json:"clan"`
	Rarity string `json:"rarity"`
	Illustrator string `json:"illustrator"`
	Effect string `json:"effect"`
	Flavor string `json:"flavor"`
}


type Card struct {
	ID       string
	CardNumberFull string
	Name *string
	Type *string
	Nation *string
	Race *string
	Grade int
	Power int
	Critical int
	Shield int
	Clan *string
	Skill *string
	Gift string
	Rarity string
	Illustrator *string
	Effect *CardText
	Flavor string
	Boons *Boon
}

func (rc *RawCard) ToCard() (*Card, error) {
	UnparsedEffect := strings.Split(strings.Replace(rc.Effect, "\n・", "・", -1), "\n")
	ParsedEffects := make([]CardText, 0)

	for i, effectLine := range UnparsedEffect {
		ParsedEffects.Append(ParseCardText{
			Description: effectLine,
			Condition: ConditionalTrigger{},
			Effect: EffectiveEffect{},
			SubEffect: nil
		})
	}

	return Card{
		ID: uuid.New().String(),
		CardNumberFull: rc.CardNumberFull,
		Name: [ rc.Name ],
		Type: strings.Split(rc.Type, "/"),
		Nation: strings.Split(rc.Nation, "/"),
		Race: strings.Split(rc.Race, "/"),
		Grade: strconv.Atoi(strings.Replace(rc.Grade, "Grade ", "", -1)),
		Power: strconv.Atoi(strings.Replace(rc.Power, "Power ", "", -1)),
		Critical: strconv.Atoi(strings.Replace(rc.Critical, "Critical ", "", -1)),
		Shield: strconv.Atoi(strings.Replace(rc.Shield, "Shield ", "", -1)),
		Clan: strings.Split(rc.Clan, "/"),
		Skill: [ rc.Skill ],
		Gift: rc.Gift,
		Rarity: rc.Rarity,
		Illustrator: strings.Split(rc.Illustrator, "/"),
		Effect: ParsedEffects,
		Flavor: rc.Flavor,
		Boons : []
	}
}