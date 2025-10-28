package player

import (
	"bufio"
	"fmt"
	"music-cli/utils"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dhowden/tag"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"golang.org/x/term"
)

var printMu sync.Mutex

type Player struct {
	// 核心播放组件
	streamer beep.StreamSeekCloser
	format   beep.Format
	ctrl     *beep.Ctrl // 新增：用于控制暂停/继续

	// 元数据
	path string
	file *os.File

	// UI组件
	pb    *progressBar
	lyric *lyrics

	// 状态管理
	isPaused  bool
	mu        sync.Mutex
	done      chan struct{}
	closeOnce sync.Once
}

func NewPlayer(path string) *Player {
	return &Player{
		path:  path,
		lyric: newLyrics(nil),
		done:  make(chan struct{}),
	}
}

func (p *Player) Init() error {
	f, err := os.Open(p.path)
	if err != nil {
		return err
	}
	p.file = f
	p.lyric = newLyrics(nil)
	p.LoadLyric()
	switch filepath.Ext(p.path) {
	case ".mp3":
		streamer, format, err := mp3.Decode(f)
		if err != nil {
			return err
		}
		p.streamer = streamer
		p.format = format
	case ".flac":
		streamer, format, err := flac.Decode(f)
		if err != nil {
			return err
		}
		p.streamer = streamer
		p.format = format
	case ".wav":
		streamer, format, err := wav.Decode(f)
		if err != nil {
			return err
		}
		p.streamer = streamer
		p.format = format
	default:
		return fmt.Errorf("unsupported audio format: %s", filepath.Ext(p.path))
	}

	p.ctrl = &beep.Ctrl{Streamer: p.streamer}
	p.isPaused = false

	return nil
}

func (p *Player) LoadLyric() {
	file, err := os.Open(p.path)
	if err != nil {
		return
	}
	defer file.Close()

	meta, err := tag.ReadFrom(file)
	if err != nil {
		return
	}
	lyricData := meta.Lyrics()
	p.lyric.parse(lyricData)
}

func (p *Player) Play() {
	if p.format.SampleRate == 0 {
		return
	}
	speaker.Init(p.format.SampleRate, p.format.SampleRate.N(time.Second/10))

	totalTime := time.Duration(p.streamer.Len()) * time.Second / time.Duration(p.format.SampleRate)
	p.pb = newProgressBar(totalTime)

	speaker.Play(beep.Seq(p.ctrl, beep.Callback(func() {
		p.closeOnce.Do(func() { close(p.done) })
	})))

	go p.displayLoop()

	<-p.done
	p.Close()
}

func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	speaker.Lock()
	p.ctrl.Paused = !p.isPaused
	speaker.Unlock()
	p.isPaused = !p.isPaused

}

func (p *Player) displayLoop() {
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go p.pb.printBar(&wg, p)
	go p.lyric.print(&wg, p)
	wg.Wait()
}

func (p *Player) Close() {
	if p.streamer != nil {
		_ = p.streamer.Close()
		p.streamer = nil
	}
	if p.file != nil {
		_ = p.file.Close()
		p.file = nil
	}
	p.closeOnce.Do(func() {
		close(p.done)
	})
}

func (p *Player) getCurrentTime() time.Duration {
	if p.streamer != nil {
		return time.Duration(p.streamer.Position()) * time.Second / time.Duration(p.format.SampleRate)
	}
	return 0
}

func handleInput(p *Player) {
	for {
		select {
		case <-p.done:
			return
		default:
			buffer := make([]byte, 1)
			n, err := os.Stdin.Read(buffer)
			if err != nil || n == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			switch buffer[0] {
			case ' ':
				p.TogglePause()
			case 'q', 'Q':
				p.Close()
				return
			}
		}
	}
}

func GetPathAndPlay() {
	for {
		fmt.Print("\033[2J\033[H")
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("请输入音乐路径：")
		scanner.Scan()
		path := scanner.Text()
		if path == "q" || path == "Q" {
			os.Exit(0)
		}
		path = strings.Trim(path, `"`)
		pathList, err := utils.GetPaths(path)
		if err != nil {
			fmt.Println("路径错误，请重新输入")
			time.Sleep(2 * time.Second)
			continue
		}

		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			continue
		}

		for _, p := range pathList {
			player := NewPlayer(p)
			if err := player.Init(); err != nil {
				continue
			}
			go handleInput(player)
			player.Play()
			player.Close()
		}

		term.Restore(int(os.Stdin.Fd()), oldState)

		fmt.Print("\033[2J\033[H")
		fmt.Print("\n播放完成，按回车继续...")
		fmt.Scanln()
	}
}
