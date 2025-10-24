package player

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dhowden/tag"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

type Player struct {
	streamer beep.StreamSeekCloser
	format   beep.Format
	path     string
	file     *os.File
	pb       *ProgressBar
	lyric    *Lyric
}

func NewPlayer(path string) *Player {
	return &Player{
		path:  path,
		lyric: NewLyric(nil),
	}
}

func (p *Player) Init() error {
	f, err := os.Open(p.path)
	if err != nil {
		return err
	}
	p.file = f
	p.lyric = NewLyric(nil)
	p.LoadLyric()
	switch filepath.Ext(p.path) {
	case ".mp3":
		streamer, format, err := mp3.Decode(f)
		fmt.Println("MP3 player initialized.")
		if err != nil {
			return err
		}
		fmt.Printf("MP3 format: %+v\n", format)
		fmt.Printf("MP3 streamer: %+v\n", streamer)
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
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	meta, err := tag.ReadFrom(file)
	if err != nil {
		fmt.Println("Failed to read metadata:", err)
		return
	}
	lyricData := meta.Lyrics()
	p.lyric.ParseLyric(lyricData)
}

func (p *Player) Play() error {
	speaker.Init(p.format.SampleRate, p.format.SampleRate.N(time.Second/10))

	totalTime := time.Duration(p.streamer.Len()) * time.Second / time.Duration(p.format.SampleRate)
	p.pb = NewProgressBar(totalTime)

	done := make(chan bool)
	speaker.Play(beep.Seq(p.streamer, beep.Callback(func() {
		fmt.Println("Playback finished.")
		done <- true
	})))
	fmt.Print(p.lyric.GetAllLyrics())
	/* go p.pb.PrintBar() */
	go p.printCurrentTest()
	<-done
	return nil
}

func (p *Player) printCurrentTest() {
	currentSample := p.streamer.Position()
	currentTime := time.Duration(currentSample) * time.Second / time.Duration(p.format.SampleRate)
	currentLyricText := p.lyric.GetCurrentLyric(currentTime)
	fmt.Printf("\r>>> %s (%v)", currentLyricText, currentTime)
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
