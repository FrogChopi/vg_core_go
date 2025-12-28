package core

import "strings"

// Condition represents a function that checks if a specific criteria is met.
type Condition func(party *Party, player *Player, source *Card) bool

// Common Conditions

// IsPhase checks if the current phase matches the target phase.
func IsPhase(phaseName string) Condition {
	return func(party *Party, player *Player, source *Card) bool {
		return strings.EqualFold(party.CurrentPhase, phaseName)
	}
}

// IsTurnPlayer checks if it is currently the player's turn.
// Note: This relies on party.Turn logic matching player index logic.
func IsTurnPlayer() Condition {
	return func(party *Party, player *Player, source *Card) bool {
		if player == nil {
			return false
		}
		// Find player index
		playerIndex := -1
		for i := range party.Players {
			if &party.Players[i] == player {
				playerIndex = i
				break
			}
		}
		// Assuming 2 players and Turn counter starting at 1 for Player 0
		if playerIndex == -1 {
			return false
		}

		// If Turn 1 -> Index 0.
		// (1 - 1) % 2 = 0.
		currentPlayerIndex := (party.Turn - 1) % len(party.Players)
		return playerIndex == currentPlayerIndex
	}
}

// HasGradeGreaterOrEqual checks if the card has a grade >= target.
func HasGradeGreaterOrEqual(grade int) Condition {
	return func(party *Party, player *Player, source *Card) bool {
		if source == nil {
			return false
		}
		return source.Grade >= grade
	}
}

// HasUnitInVanguard checks if the player has a unit in Vanguard circle.
func HasUnitInVanguard(paramName interface{}) Condition {
	// Example of a condition that might take specific params,
	// though usually we check the player's board state.
	// paramName is just a placeholder here if we wanted to check for a specific Name.
	return func(party *Party, player *Player, source *Card) bool {
		return player.Vanguard.TopCard != nil
	}
}
