package main

import (
	"fmt"
	"vg_core_go/internal/core"
)

func defaultGame() {
	deck1, err1 := core.ParseDeckFile("decks/KT_Starter.md")
	deck2, err2 := core.ParseDeckFile("decks/LM_Starter.md")

	if err1 != nil || err2 != nil {
		fmt.Println(err1, err2)
		return
	}

	// core.PrintDeck(deck1)
	// core.PrintDeck(deck2)

	party := core.InitParty([]*core.Deck{deck1, deck2})

	// core.PrintParty(party)

	core.InitGame(party, "")

	core.PrintParty(party)
}

func main() {
	core.StartServer("8080")
	// defaultGame()
}
