package main

import (
	"fmt"
	"vg_core_go/internal/core"
)

func main() {
	deck, err := core.ParseDeckFile("decks/KT_Starter.md")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(deck)
}
