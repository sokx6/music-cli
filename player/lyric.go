package player

import (
	"fmt"
	"music-cli/utils"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Lyric struct {
	LyricLines   []LyricLine
	currentIndex int
}

type LyricLine struct {
	OriginalLine   lyricLine
	TranslatedLine lyricLine
}

type lyricLine struct {
	Time  time.Duration
	Words []Word
	Text  string
}

type Word struct {
	Time time.Duration
	Text string
}

func NewLyric(lines []LyricLine) *Lyric {
	return &Lyric{
		LyricLines:   lines,
		currentIndex: -1,
	}
}

func (l *Lyric) ParseLyric(rawLyrics string) {
	currentLine := lyricLine{}
	lastLine := lyricLine{}
	for _, line := range strings.Split(rawLyrics, "\n") {

		lyricLine, err := ParseLine(line)
		if err != nil {
			continue
		}
		lastLine = currentLine
		lastLine = currentLine
		currentLine = lyricLine
		if currentLine.Time == lastLine.Time {
			l.LyricLines = append(l.LyricLines, LyricLine{
				OriginalLine:   lastLine,
				TranslatedLine: currentLine,
			})
		}

	}

}

func ParseLine(line string) (lyricLine, error) {
	var lyricLine lyricLine
	pattern := `\[(\d{2}):(\d{2})\.(\d{2,3})\]([^[]*)`
	regex := regexp.MustCompile(pattern)
	matches := regex.FindAllStringSubmatch(line, -1)
	if len(matches) < 1 {
		return lyricLine, fmt.Errorf("no match found")
	} else if len(matches) == 1 {
		word, err := parseWord(matches[0])
		if err != nil {
			return lyricLine, err
		}
		lyricLine.Time = word.Time
		lyricLine.Words = []Word{word}
		lyricLine.Text = word.Text
		return lyricLine, nil
	}

	text := ""
	for i, match := range matches {
		word, err := parseWord(match)
		if err != nil {
			return lyricLine, err
		}
		if strings.TrimSpace(word.Text) == "" {
			continue
		}
		if i == 0 {
			lyricLine.Time = word.Time
		}
		lyricLine.Words = append(lyricLine.Words, word)
		text += word.Text
	}
	lyricLine.Text = text
	return lyricLine, nil
}

func parseWord(wordInfo []string) (Word, error) {
	min, err := time.ParseDuration(wordInfo[1] + "m")
	if err != nil {
		return Word{}, err
	}
	sec, err := time.ParseDuration(wordInfo[2] + "s")
	if err != nil {
		return Word{}, err
	}
	ms, err := time.ParseDuration(wordInfo[3] + "ms")
	if err != nil {
		return Word{}, err
	}
	return Word{
		Time: min + sec + ms,
		Text: wordInfo[4],
	}, nil
}

func (l *Lyric) GetAllLyrics() []LyricLine {
	return l.LyricLines
}

func (l *Lyric) printLyric(wg *sync.WaitGroup) {
	defer wg.Done()
	start := time.Now()
	var lastLyric string
	var currentLyric string
	var nextLyric string
	for i, ly := range l.LyricLines {
		currentLyric = ly.OriginalLine.Text
		if i < len(l.LyricLines)-1 {
			nextLyric = l.LyricLines[i+1].OriginalLine.Text
		}
		if i > 0 {
			lastLyric = l.LyricLines[i-1].OriginalLine.Text
		}
		target := start.Add(ly.OriginalLine.Time)
		now := time.Now()

		timer := time.NewTimer(target.Sub(now))
		<-timer.C

		printMu.Lock()
		fmt.Print("\033[4;1f")
		fmt.Print("\033[2K")
		fmt.Print(utils.Center(lastLyric))
		fmt.Print("\033[6;1f")
		fmt.Print("\033[2K")
		fmt.Print(utils.Center("➣ " + currentLyric))
		fmt.Print("\033[8;1f")
		fmt.Print("\033[2K")
		fmt.Print(utils.Center(nextLyric))
		printMu.Unlock()
	}
}
