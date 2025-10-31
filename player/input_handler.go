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
const welcomeMessage = `欢迎使用music-cli音乐播放器

基本操作
播放 / 暂停：空格 (Space)
上一首： -
下一首： +
退出播放返回目录：q / Q

菜单与浏览
输入编号：播放对应音乐或进入目录
下一页 / 上一页：输入 + / -
跳转到指定页：pN（例如 p2）
切换盘符：c:（c是一个字母）
上一级目录：..
播放当前页全部：0 / -0 / +0
当前目录全部播放（递归）：a
递归随机播放：ar
随机播放当前页：0r
随机播放当前页单首：r`

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
		fmt.Print("当前目录为空")
		time.Sleep(1 * time.Second)
		if root == "/" || root == "\\" || len(filepath.Dir(root)) >= len(root) {
			pageChannel <- pageChange{signal: toHomeSignal}
			return nil
		}
		pageChannel <- pageChange{signal: toMenuSignal, root: filepath.Dir(root), page: 1}
		return nil
	}
	if err != nil {
		fmt.Println("错误:", err)
		return err
	}
	utils.PrintPathInfo(root, page)
	fmt.Print("请输入音乐或目录编号（q键回到主菜单）：")
	var input string
	var index int
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input = strings.Trim(scanner.Text(), " \t\n\r'\"")
	if input == "" {
		os.Exit(0)
	}
	needReturn, err := handleMenuInput(root, page, input, files)
	if needReturn || err != nil {
		return err
	}
	for index, err = strconv.Atoi(input); err != nil || index < -2; {
		fmt.Print("输入无效，请重新输入编号（q键回到主菜单）：")
		scanner.Scan()
		input = strings.Trim(scanner.Text(), " \t\n\r'\"")
		if input == "" {
			os.Exit(0)
		}
		needReturn, err = handleMenuInput(root, page, input, files)
		if needReturn || err != nil {
			return err
		}
		index, err = strconv.Atoi(input)
	}
	if index <= len(files) && index > 0 {
		player := NewPlayer(files[index-1], 1)
		handlePlayInput(root, 0, page, []*Player{player})
		return nil
	} else if index <= len(files)+len(dir) {
		pageChannel <- pageChange{signal: toMenuSignal, root: dir[index-len(files)-1], page: 1}
		return nil
	}
	pageChannel <- pageChange{signal: toMenuSignal, root: root}
	return nil
}

func handleHomeInput() {
	fmt.Print("\033[2J\033[H")
	fmt.Println(welcomeMessage)
	fmt.Println()
	fmt.Print("请输入音乐路径(回车或q键直接退出)：")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	path := strings.Trim(scanner.Text(), " \t\n\r'\"")
	if path == "q" || path == "Q" || path == "" {
		os.Exit(0)
	}
	info, err := os.Lstat(path)
	for err != nil {
		fmt.Print("请输入音乐路径(回车或q键直接退出)：")
		scanner.Scan()
		path = strings.Trim(scanner.Text(), " \t\n\r'\"")
		if path == "q" || path == "Q" || path == "" {
			os.Exit(0)
		}
		info, err = os.Lstat(path)
	}
	if !info.IsDir() {
		player := NewPlayer(path, 1)
		handlePlayInput(filepath.Dir(path), 0, 1, []*Player{player})
		return
	}
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
		fmt.Print("aaaaaaaaaaa")
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
	case "..":
		parentDir := filepath.Dir(root)
		pageChannel <- pageChange{signal: toMenuSignal, root: parentDir, page: 1}
		return true, nil
	}
	if input == "" {
		return false, nil
	}
	if input[:1] == "p" && len(input) > 1 {
		page, err := strconv.Atoi(input[1:])
		if err != nil || page < 1 {
			return false, err
		}
		pageChannel <- pageChange{signal: toMenuSignal, root: root, page: page}
		return true, nil
	}
	if len(input) == 2 && input[1:2] == ":" {
		pageChannel <- pageChange{signal: toMenuSignal, root: input + "\\", page: 1}
		return true, nil
	}
	return false, nil
}
