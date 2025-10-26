package player

import (
	"fmt"
	"music-cli/utils"
	"regexp"
	"strconv"
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
	hasLast := false
	var last lyricLine

	for _, line := range strings.Split(rawLyrics, "\n") {
		lyricLine, err := ParseLine(line)
		if err != nil {
			continue
		}

		if !hasLast {
			// 保存第一行，等待可能的同时间戳的翻译行
			last = lyricLine
			hasLast = true
			continue
		}

		// 如果时间相同，视为原文+翻译配对
		if lyricLine.Time == last.Time {
			l.LyricLines = append(l.LyricLines, LyricLine{
				OriginalLine:   last,
				TranslatedLine: lyricLine,
			})
			hasLast = false
		} else {
			// 否则把上一次单独作为原文行保存，当前行作为下一次的 last
			l.LyricLines = append(l.LyricLines, LyricLine{
				OriginalLine: last,
			})
			last = lyricLine
			hasLast = true
		}
	}

	// 循环结束后，如果还有未配对的最后一行，作为单独原文行加入
	if hasLast {
		l.LyricLines = append(l.LyricLines, LyricLine{
			OriginalLine: last,
		})
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
	// 处理小数部分：如果是两位（如 "34"），表示 340ms；如果三位则直接是 ms
	msStr := wordInfo[3]
	msInt, err := strconv.Atoi(msStr)
	if err != nil {
		return Word{}, err
	}
	if len(msStr) == 2 {
		msInt *= 10
	}
	ms := time.Duration(msInt) * time.Millisecond

	return Word{
		Time: min + sec + ms,
		Text: wordInfo[4],
	}, nil
}

func (l *Lyric) GetAllLyrics() []LyricLine {
	return l.LyricLines
}

func (l *Lyric) printLyric(wg *sync.WaitGroup, player *Player) {
	defer wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		currentTime := player.getCurrentTime()
		l.printCurrentLyric(currentTime)
	}
}

func (l *Lyric) getCurrentLyric(currentTime time.Duration) (int, LyricLine) {
	n := len(l.LyricLines)
	if n == 0 {
		return -1, LyricLine{}
	}
	// 在第一行之前
	if currentTime < l.LyricLines[0].OriginalLine.Time {
		return -1, LyricLine{}
	}
	// 在某行与下一行之间，返回当前行索引
	for i := 0; i < n-1; i++ {
		if currentTime >= l.LyricLines[i].OriginalLine.Time && currentTime < l.LyricLines[i+1].OriginalLine.Time {
			return i, l.LyricLines[i]
		}
	}
	// 在最后一行及之后
	return n - 1, l.LyricLines[n-1]
}

func (l *Lyric) printCurrentLyric(currentTime time.Duration) {
	var lastOriginalLine lyricLine
	var currentOriginalLine lyricLine
	var nextOriginalLine lyricLine
	var lastTranslatedLine lyricLine
	var currentTranslatedLine lyricLine
	var nextTranslatedLine lyricLine

	lIndex, currentLine := l.getCurrentLyric(currentTime)

	// 当前行（可能为零值，如果 lIndex == -1）
	if lIndex >= 0 {
		currentOriginalLine = currentLine.OriginalLine
		currentTranslatedLine = l.LyricLines[lIndex].TranslatedLine
	}

	if lIndex-1 >= 0 {
		lastOriginalLine = l.LyricLines[lIndex-1].OriginalLine
		lastTranslatedLine = l.LyricLines[lIndex-1].TranslatedLine
	}
	if lIndex+1 < len(l.LyricLines) {
		nextOriginalLine = l.LyricLines[lIndex+1].OriginalLine
		nextTranslatedLine = l.LyricLines[lIndex+1].TranslatedLine
	}

	// 使用当前原文计算当前字索引（如果没有当前原文，getCurrentWord 会返回 -1）
	wIndex, _ := getCurrentWord(currentOriginalLine, currentTime)

	printMu.Lock()
	defer printMu.Unlock()
	fmt.Print("\033[4;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center(lastOriginalLine.Text))

	fmt.Print("\033[5;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center(lastTranslatedLine.Text))

	fmt.Print("\033[7;1H")
	fmt.Print("\033[2K")
	plain := "➣ " + l.getWordTextPlain(currentOriginalLine, wIndex)
	colored := "\x1b[34m➣ " + l.getWordText(currentOriginalLine, wIndex)
	centered := utils.Center(plain)
	lineToPrint := strings.Replace(centered, plain, colored, 1)
	fmt.Print(lineToPrint)

	fmt.Print("\033[8;1H")
	fmt.Print("\033[2K")
	// 这里应打印当前行的翻译（之前错误地打印了 lastTranslatedLine）
	fmt.Print(utils.Center(currentTranslatedLine.Text))

	fmt.Print("\033[10;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center(nextOriginalLine.Text))

	fmt.Print("\033[11;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center(nextTranslatedLine.Text))
}

func getCurrentWord(lyricLine lyricLine, currentTime time.Duration) (int, Word) {
	if len(lyricLine.Words) == 0 {
		return -1, Word{}
	}
	for i, w := range lyricLine.Words {
		if currentTime < w.Time {
			// 如果还没到第一个字，返回 -1 表示没有已播放的字
			if i == 0 {
				return -1, Word{}
			}
			return i - 1, lyricLine.Words[i-1]
		}
	}
	// 已经超过最后一个字，返回最后一个字
	last := len(lyricLine.Words) - 1
	return last, lyricLine.Words[last]
}

func (l *Lyric) getWordText(line lyricLine, index int) string {
	if index < 0 || index >= len(line.Words) {
		return "\x1b[0m"
	}
	playedWords := "\x1b[34m"
	unPlayedWords := "\x1b[30;1m█"
	for i := 0; i <= index; i++ {
		playedWords += line.Words[i].Text
	}
	for i := index + 1; i < len(line.Words); i++ {
		unPlayedWords += line.Words[i].Text
	}
	// 末尾重置颜色，避免影响居中和后续输出
	return playedWords + unPlayedWords + "\x1b[0m"
}

func (l *Lyric) getWordTextPlain(line lyricLine, index int) string {
	// 返回不带 ANSI 颜色码的可视文本，用于计算居中
	if len(line.Words) == 0 {
		return line.Text
	}
	if index < 0 {
		return line.Text
	}
	if index >= len(line.Words) {
		// 全部为已播放状态
		index = len(line.Words) - 1
	}
	played := "█"
	unPlayed := "█"
	for i := 0; i <= index; i++ {
		played += line.Words[i].Text
	}
	for i := index + 1; i < len(line.Words); i++ {
		unPlayed += line.Words[i].Text
	}
	return played + unPlayed
}
