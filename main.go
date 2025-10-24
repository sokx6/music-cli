package main

import (
	"fmt"
	"log"
	"music-cli/player"
)

func main() {
	player := player.NewPlayer("test.mp3")
	if err := player.Init(); err != nil {
		log.Fatal(err)
	}
	defer player.Close()
	fmt.Println("Playing audio...")
	if err := player.Play(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Playback finished.")
}
