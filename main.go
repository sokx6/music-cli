package main

import (
	"fmt"
	"log"
	"music-cli/player"
)

func main() {
	fmt.Println("请输出音乐路路径：")
	var path string
	fmt.Scanln(&path)
	player := player.NewPlayer(path)
	if err := player.Init(); err != nil {
		log.Fatal(err)
	}
	defer player.Close()
	if err := player.Play(); err != nil {
		log.Fatal(err)
	}
}
