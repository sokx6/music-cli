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
			case '+':
				currentPlayer.Close()
				currentIndex = (currentIndex + 1) % len(plist)
				currentPlayer = plist[currentIndex]
				currentPlayer.Init()
				go currentPlayer.Play()
				doneCh = currentPlayer.done
			case '-':
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

func handleMenu(root string, page int) error {
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
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input = strings.TrimSpace(scanner.Text())
	needReturn, err := handleMenuInput(root, page, input, files)
	if needReturn || err != nil {
		return err
	}
	for index, err = strconv.Atoi(input); err != nil || index < -2; {
		fmt.Print("输入无效，请重新输入编号：")
		scanner.Scan()
		input = scanner.Text()
		needReturn, err = handleMenuInput(root, page, input, files)
		if needReturn || err != nil {
			return err
		}
		input = strings.TrimSpace(input)
		index, err = strconv.Atoi(input)
	}
	if index <= len(files) && index > 0 {
		player := NewPlayer(files[index-1], 1)
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
	for err != nil {
		fmt.Print("请输入音乐路径：")
		scanner.Scan()
		path = scanner.Text()
		if path == "q" || path == "Q" {
			os.Exit(0)
		}
		info, err = os.Lstat(path)
	}
	if !info.IsDir() {
		player := NewPlayer(path, 1)
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
			go handleMenu(pc.root, pc.page)
		case toHomeSignal:
			go handleHomeInput()
		case exitSignal:
			return
		}
	}
}

func handleMenuInput(root string, page int, input string, files []string) (bool, error) {
	switch input {
	case "q", "Q":
		pageChannel <- pageChange{signal: toHomeSignal, root: root}
		return true, nil
	case "+":
		pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page + 1}
		return true, nil
	case "-":
		if page <= 1 {
			pageChannel <- pageChange{signal: toMenuSignal, root: root, page: 1}
		} else {
			pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page - 1}
		}
		return true, nil
	case "r", "R":
		players := randomPlayer(getPlayerList(files))
		var singlePlayerList []*Player
		singlePlayerList = append(singlePlayerList, players[0])
		handlePlayInput(root, 0, page, singlePlayerList)
		return true, nil
	case "a", "A":
		pathStrList, err := utils.WalkDir(root)
		if err != nil {
			fmt.Println("错误:", err)
			return true, err
		}
		players := getPlayerList(pathStrList)
		handlePlayInput(root, 0, page, players)
		return true, nil
	case "ar", "aR":
		pathStrList, err := utils.WalkDir(root)
		if err != nil {
			fmt.Println("错误:", err)
			return true, err
		}
		players := randomPlayer(getPlayerList(pathStrList))
		handlePlayInput(root, 0, page, players)
		return true, nil
	case "0r", "0R":
		players := randomPlayer(getPlayerList(files))
		handlePlayInput(root, 0, page, players)
		return true, nil
	case "0", "-0", "+0":
		players := getPlayerList(files)
		handlePlayInput(root, 0, page, players)
		return true, nil
	}
	if input[:1] == "p" && len(input) > 1 {
		page, err := strconv.Atoi(input[1:])
		if err != nil || page < 1 {
			return false, err
		}
		pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page}
		return true, nil
	}
	return false, nil
}
