package player

import (
	"bufio"
	"fmt"
	"music-cli/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	exitSignal   = 0
	toMenuSignal = 1
	toHomeSignal = 2
)

type pageChange struct {
	signal int
	root   string
	page   int
}

var pageChannel = make(chan pageChange)

func handlePlayInput(root string, start int, page int, plist []*Player) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	if err != nil {
		return
	}

	currentIndex := start
	currentPlayer := plist[currentIndex]

	currentPlayer.Init()
	go currentPlayer.Play()

	bytesCh := make(chan byte, 16)

	readerQuit := make(chan struct{})

	go func() {
		for {
			select {
			case <-readerQuit:
				return
			default:
				buf := make([]byte, 1)
				n, err := os.Stdin.Read(buf)
				if err != nil || n == 0 {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				bytesCh <- buf[0]
			}
		}
	}()

	doneCh := currentPlayer.done

	for {
		select {
		case b := <-bytesCh:
			switch b {
			case ' ':
				currentPlayer.TogglePause()
			case 'd', 'D':
				currentPlayer.Close()
				currentIndex = (currentIndex + 1) % len(plist)
				currentPlayer = plist[currentIndex]
				currentPlayer.Init()
				go currentPlayer.Play()
				doneCh = currentPlayer.done
			case 'w', 'W':
				fmt.Println("Previous track")
				currentPlayer.Close()
				currentIndex = (currentIndex - 1 + len(plist)) % len(plist)
				currentPlayer = plist[currentIndex]
				currentPlayer.Init()
				go currentPlayer.Play()
				doneCh = currentPlayer.done
			case 'q', 'Q':
				currentPlayer.Close()
				close(readerQuit)
				pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page}
				return
			}
		case <-doneCh:
			currentIndex = (currentIndex + 1) % len(plist)
			currentPlayer = plist[currentIndex]
			currentPlayer.Init()
			go currentPlayer.Play()
			doneCh = currentPlayer.done
		}
	}
}

func handleMenuInput(root string, page int) error {
	fmt.Print("\033[2J\033[H")
	files, dir, err := utils.ListDir(root)
	if len(files) == 0 && len(dir) == 0 {
		fmt.Println("当前目录为空")
		time.Sleep(1 * time.Second)
		pageChannel <- pageChange{signal: toHomeSignal}
		return nil
	}
	if err != nil {
		fmt.Println("错误:", err)
		return err
	}
	utils.PrintPathInfo(root, page)
	fmt.Print("请输入音乐或目录编号：")
	var input string
	var index int
	fmt.Scan(&input)
	switch input {
	case "q", "Q":
		pageChannel <- pageChange{signal: toHomeSignal, root: root}
		return nil
	case "d", "D":
		pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page + 1}
		return nil
	case "a", "A":
		if page <= 1 {
			pageChannel <- pageChange{signal: toMenuSignal, root: root, page: 1}
		} else {
			pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page - 1}
		}
		return nil
	}
	for index, err = strconv.Atoi(input); err != nil || index < -1; {
		fmt.Print("输入无效，请重新输入编号：")
		fmt.Scan(&input)
		switch input {
		case "q", "Q":
			pageChannel <- pageChange{signal: toHomeSignal, root: root}
			return nil
		case "d", "D":
			pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page + 1}
			return nil
		case "a", "A":
			if page <= 1 {
				pageChannel <- pageChange{signal: toMenuSignal, root: root, page: 1}
			} else {
				pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page - 1}
			}
			return nil
		}
		input := strings.TrimSpace(input)
		index, err = strconv.Atoi(input)
	}

	if index == -1 {
		pathStrList, err := utils.WalkDir(root)
		if err != nil {
			fmt.Println("错误:", err)
			return err
		}
		players := getPlayerList(pathStrList)
		handlePlayInput(root, 0, page, players)
		return nil
	} else if index == 0 {
		players := getPlayerList(files)
		handlePlayInput(root, 0, page, players)
		return nil
	} else if index <= len(files) {
		player := NewPlayer(files[index-1])
		handlePlayInput(root, 0, page, []*Player{player})
		return nil
	} else if index <= len(files)+len(dir) {
		pageChannel <- pageChange{signal: toMenuSignal, root: dir[index-len(files)-1]}
		return nil
	}
	pageChannel <- pageChange{signal: toMenuSignal, root: root}
	return nil
}

func handleHomeInput() {
	fmt.Print("\033[2J\033[H")
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("请输入音乐路径：")
	scanner.Scan()
	path := scanner.Text()
	if path == "q" || path == "Q" {
		os.Exit(0)
	}
	info, err := os.Lstat(path)
	if !info.IsDir() && err == nil {
		player := NewPlayer(path)
		handlePlayInput(filepath.Dir(path), 1, 1, []*Player{player})
		return
	}
	path = strings.Trim(path, `"`)
	pageChannel <- pageChange{signal: toMenuSignal, root: path, page: 1}
}

func PageController() {
	go handleHomeInput()
	for {
		switch pc := <-pageChannel; pc.signal {
		case toMenuSignal:
			go handleMenuInput(pc.root, pc.page)
		case toHomeSignal:
			go handleHomeInput()
		case exitSignal:
			return
		}
	}
}
