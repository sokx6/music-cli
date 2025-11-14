package player

import (
	"fmt"
	"math/rand"
	"music-cli/utils"
	"os"
	"path/filepath"
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

/* defaultMetadata 提供了一个默认的元数据实现
 * 这我只能说就是依托
 * 很难想象我为什么会用这种方法
 */
type defaultMetadata struct {
}

func (d *defaultMetadata) Format() tag.Format {
	return tag.UnknownFormat
}
func (d *defaultMetadata) Title() string {
	return "Unknown Title"
}
func (d *defaultMetadata) Artist() string {
	return "Unknown Artist"
}
func (d *defaultMetadata) Album() string {
	return "Unknown Album"
}
func (d *defaultMetadata) Year() int {
	return 0
}
func (d *defaultMetadata) Genre() string {
	return "Unknown Genre"
}
func (d *defaultMetadata) AlbumArtist() string {
	return "Unknown Album Artist"
}
func (d *defaultMetadata) Composer() string {
	return "Unknown Composer"
}
func (d *defaultMetadata) FileType() tag.FileType {
	return tag.UnknownFileType
}
func (d *defaultMetadata) Lyrics() string {
	return ""
}
func (d *defaultMetadata) Track() (int, int) {
	return 0, 0
}
func (d *defaultMetadata) Disc() (int, int) {
	return 0, 0
}
func (d *defaultMetadata) Picture() *tag.Picture {
	return nil
}
func (d *defaultMetadata) Raw() map[string]interface{} {
	return nil
}
func (d *defaultMetadata) Comment() string {
	return ""
}

type Player struct {
	id int
	// 核心播放组件
	streamer beep.StreamSeekCloser
	format   beep.Format
	ctrl     *beep.Ctrl // 新增：用于控制暂停/继续

	// 元数据
	path     string
	file     *os.File
	metadata tag.Metadata
	// UI组件
	pb    *progressBar
	lyric *lyrics

	// 状态管理
	isPaused  bool
	mu        sync.Mutex
	done      chan struct{}
	closeOnce sync.Once
}

func NewPlayer(path string, id int) *Player {
	return &Player{
		path:  path,
		lyric: newLyrics(nil),
		done:  make(chan struct{}),
		id:    id,
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
		p.metadata = &defaultMetadata{}
		return
	}
	if meta == nil {
		p.metadata = &defaultMetadata{}
		return
	}
	p.metadata = meta
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

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second)/10)

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
	fmt.Print(utils.Center(fmt.Sprintf("[%d]: %s - %s", p.id, p.metadata.Artist(), p.metadata.Title())))
	wg := sync.WaitGroup{}
	wg.Add(3)
	var clearChan = make(chan struct{})
	go clearScreen(&wg, p, clearChan)
	go p.pb.printBar(&wg, p)
	go p.lyric.print(&wg, p, clearChan)
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

func getPlayerList(paths []string) []*Player {
	var players []*Player
	for i, path := range paths {
		player := NewPlayer(path, i+1)
		players = append(players, player)
	}
	return players
}
func randomPlayer(players []*Player) []*Player {
	if len(players) == 0 {
		return nil
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})
	for i, player := range players {
		player.id = i + 1
	}
	return players
}

func clearScreen(wg *sync.WaitGroup, player *Player, clearChan chan struct{}) {
	defer wg.Done()

	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
	currentWidth, currentHeight, _ := term.GetSize(int(os.Stdout.Fd()))
	var lastWidth, lastHeight int

	for {
		select {
		case <-player.done:
			return
		case <-ticker.C:

			lastWidth, lastHeight = currentWidth, currentHeight
			currentWidth, currentHeight, _ = term.GetSize(int(os.Stdout.Fd()))
			if currentWidth != lastWidth || currentHeight != lastHeight {
				printMu.Lock()
				fmt.Print("\033[2J\033[H")
				printMu.Unlock()
				clearChan <- struct{}{}
			}

		}
	}
}
