package player

import (
	"fmt"
	"strings"
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

func (l *Lyric) GetCurrentLyric(currentTime time.Duration) string {
	// 如果歌词为空，直接返回
	if len(l.LyricLines) == 0 {
		l.currentIndex = -1
		return ""
	}

	// 如果索引还没初始化，或者当前时间小于了当前索引指向的时间（比如用户拖动进度条到开头），则重置索引
	if l.currentIndex == -1 || currentTime < l.LyricLines[l.currentIndex].Time {
		l.currentIndex = 0
	}

	// 从当前索引开始，向后查找，直到找到第一个时间大于当前时间的歌词行
	// 那么我们要找的歌词就是这一行的前一行
	for l.currentIndex < len(l.LyricLines) && currentTime >= l.LyricLines[l.currentIndex].Time {
		l.currentIndex++
	}

	// 如果索引超出了范围，说明歌曲已经播完所有歌词
	if l.currentIndex == 0 {
		return "" // 还没到第一句歌词
	}

	// 返回前一句歌词，它就是当前时间点应该播放的歌词
	return l.LyricLines[l.currentIndex-1].Text
}

func (l *Lyric) ParseLyric(rawLyrics string) {
	fmt.Printf("Parsing raw lyrics:%s\n", rawLyrics)

	for _, line := range strings.Split(rawLyrics, "\n") {
		duration := time.Duration(0)
		fmt.Println("Parsing line:", line)
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
			fmt.Println("Failed to parse duration:", err)
			continue
		}
		duration = time.Duration(minute)*time.Minute + time.Duration(second)*time.Second + time.Duration(millisecond)*time.Millisecond
		fmt.Printf("Parsed line - Time: %v, Text: %s\n", duration, text)
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
