package main

import (
	"log"

	"github.com/nint8835/8-grack/bot"
)

func main() {
	err := bot.Start()

	if err != nil {
		log.Fatalf("Error starting bot: %s", err)
	}
}
