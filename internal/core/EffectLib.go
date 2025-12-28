package core

import "fmt"

// EffectAction represents the execution of an effect.
type EffectAction func(party *Party, player *Player, source *Card)

// Common Effects

// DrawEffect returns an effect that draws 'count' cards.
func DrawEffect(count int) EffectAction {
	return func(party *Party, player *Player, source *Card) {
		fmt.Printf("Effect: Drawing %d card(s) for Player\n", count)
		draw(player, count)
	}
}

// PowerUpEffect returns an effect that increases power of the source card (if active).
func PowerUpEffect(amount int) EffectAction {
	return func(party *Party, player *Player, source *Card) {
		if source != nil {
			fmt.Printf("Effect: Power +%d to %s\n", amount, source.Name)
			source.Power += amount
			// Note: In a real implementation this would likely be a temporary +Power on the Circle/Game State,
			// not modifying the base card struct permanently unless intended.
			// But for this simplified version, we modify the struct.
		}
	}
}

// RetireUnitEffect (Placeholder)
func RetireUnitEffect() EffectAction {
	return func(party *Party, player *Player, source *Card) {
		fmt.Println("Effect: Retire Unit logic goes here")
	}
}
