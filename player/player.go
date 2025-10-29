package player

import (
	"bufio"
	"fmt"
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

	p.done = make(chan struct{})
	p.closeOnce = sync.Once{}

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
	p.mu.Lock()
	format := p.format
	streamer := p.streamer
	ctrl := p.ctrl
	done := p.done
	p.mu.Unlock()

	if format.SampleRate == 0 || streamer == nil || ctrl == nil || done == nil {
		return
	}

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	totalTime := time.Duration(streamer.Len()) * time.Second / time.Duration(format.SampleRate)
	p.pb = newProgressBar(totalTime)

	speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
		p.closeOnce.Do(func() { close(done) })
	})))

	go p.displayLoop()

	<-done
	p.Close()
}

func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl == nil {
		return
	}

	speaker.Lock()
	p.ctrl.Paused = !p.isPaused
	speaker.Unlock()
	p.isPaused = !p.isPaused

}

func (p *Player) displayLoop() {
	fmt.Print("\x1b[?25l")
	fmt.Print("\033[2J\033[H")
	defer fmt.Print("\x1b[?25h")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go p.pb.printBar(&wg, p)
	go p.lyric.print(&wg, p)
	wg.Wait()
}

func (p *Player) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer != nil {
		_ = p.streamer.Close()
		p.streamer = nil
	}
	if p.file != nil {
		_ = p.file.Close()
		p.file = nil
	}
	p.closeOnce.Do(func() {
		if p.done != nil {
			close(p.done)
		}
	})
}

func (p *Player) getCurrentTime() time.Duration {
	if p.streamer != nil {
		return time.Duration(p.streamer.Position()) * time.Second / time.Duration(p.format.SampleRate)
	}
	return 0
}

func GetPathAndPlay() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("请输入音乐路径：")
	scanner.Scan()
	path := scanner.Text()
	if path == "q" || path == "Q" {
		os.Exit(0)
	}
	path = strings.Trim(path, `"`)
	handleMenuInput(path, 1)
}

func getPlayerList(paths []string) []*Player {
	var players []*Player
	for _, path := range paths {
		player := NewPlayer(path)
		players = append(players, player)
	}
	return players
}
