package main

import (
	"bufio"
	"fmt"
	"log"
	"music-cli/player"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("请输入音乐路径：")
	scanner.Scan()
	path := scanner.Text()
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
