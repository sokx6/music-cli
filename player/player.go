package player

import (
	"fmt"
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
)

var printMu sync.Mutex

type Player struct {
	streamer beep.StreamSeekCloser
	format   beep.Format
	path     string
	file     *os.File
	pb       *progressBar
	lyric    *lyrics
}

func NewPlayer(path string) *Player {
	return &Player{
		path:  path,
		lyric: newLyrics(nil),
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

func (p *Player) Play() error {
	speaker.Init(p.format.SampleRate, p.format.SampleRate.N(time.Second/10))

	totalTime := time.Duration(p.streamer.Len()) * time.Second / time.Duration(p.format.SampleRate)
	p.pb = newProgressBar(totalTime)

	done := make(chan bool)
	speaker.Play(beep.Seq(p.streamer, beep.Callback(func() {
		done <- true
	})))
	go p.displayLoop()
	<-done
	return nil
}

func (p *Player) displayLoop() {
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go p.pb.printBar(&wg)
	go p.lyric.print(&wg, p)
	wg.Wait()
}

func (p *Player) Close() error {
	if p.streamer != nil {
		_ = p.streamer.Close()
		p.streamer = nil
	}
	if p.file != nil {
		_ = p.file.Close()
		p.file = nil
	}
	return nil
}

func (p *Player) getCurrentTime() time.Duration {
	if p.streamer != nil {
		return time.Duration(p.streamer.Position()) * time.Second / time.Duration(p.format.SampleRate)
	}
	return 0
}
