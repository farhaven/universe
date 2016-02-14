package main

import (
	"log"

	"./orrery"
	"./ui"
)

func main() {
	o := orrery.New()

	width, height := 1024, 768
	ctx := ui.NewDrawContext(width, height, o)

	log.Println(`waiting for ui to shut down`)
	ctx.WaitForShutdown()
}
