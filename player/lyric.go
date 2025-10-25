package player

import (
	"fmt"
	"music-cli/utils"
	"strings"
	"sync"
	"time"
)

type Lyric struct {
	LyricLines   []LyricLine
	currentIndex int
}

type LyricLine struct {
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

	for _, line := range strings.Split(rawLyrics, "\n") {
		duration := time.Duration(0)
		if len(line) < 10 || line[0] != '[' || line[9] != ']' {
			continue
		}

		// 解析时间和歌词内容
		timeStr := line[1:9]
		text := line[10:]
		// 将时间字符串转换为 time.Duration
		var minute, second, millisecond int
		_, err := fmt.Sscanf(timeStr, "%02d:%02d.%03d", &minute, &second, &millisecond)
		if err != nil {
			/* fmt.Println("Failed to parse duration:", err) */
			continue
		}
		duration = time.Duration(minute)*time.Minute + time.Duration(second)*time.Second + time.Duration(millisecond)*time.Millisecond
		// 添加到歌词对象中
		l.LyricLines = append(l.LyricLines, LyricLine{
			Time: duration,
			Text: text,
		})
	}
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
		currentLyric = ly.Text
		if i < len(l.LyricLines)-1 {
			nextLyric = l.LyricLines[i+1].Text
		}
		if i > 0 {
			lastLyric = l.LyricLines[i-1].Text
		}
		target := start.Add(ly.Time)
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
