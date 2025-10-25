package main

import (
	"fmt"
	"log"
	"music-cli/player"
)

func main() {
	fmt.Print("请输入音乐路径：")
	var path string
	fmt.Scanln(&path)
	fmt.Print("\033[2J\033[H")
	player := player.NewPlayer(path)
	if err := player.Init(); err != nil {
		log.Fatal(err)
	}
	defer player.Close()
	if err := player.Play(); err != nil {
		log.Fatal(err)
	}
}
