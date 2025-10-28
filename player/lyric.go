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

// lyrics 管理解析后的歌词行对（原文 + 译文）
type lyrics struct {
	pairs        []lyricPair
	currentIndex int
}

// lyricPair 表示同一时间戳的原文与译文行
type lyricPair struct {
	Original   lyricLine
	Translated lyricLine
}

type lyricLine struct {
	Time  time.Duration
	Words []word
	Text  string
}

type word struct {
	Time time.Duration
	Text string
}

func newLyrics(pairs []lyricPair) *lyrics {
	return &lyrics{
		pairs:        pairs,
		currentIndex: -1,
	}
}

func (l *lyrics) parse(rawLyrics string) {
	hasLast := false
	var last lyricLine

	for _, line := range strings.Split(rawLyrics, "\n") {
		parsed, err := parseLine(line)
		if err != nil {
			continue
		}

		if !hasLast {
			// 保存第一行，等待可能的同时间戳的翻译行
			last = parsed
			hasLast = true
			continue
		}

		// 如果时间相同，视为原文+翻译配对
		if parsed.Time == last.Time {
			l.pairs = append(l.pairs, lyricPair{
				Original:   last,
				Translated: parsed,
			})
			hasLast = false
		} else {
			// 否则把上一次单独作为原文行保存，当前行作为下一次的 last
			l.pairs = append(l.pairs, lyricPair{
				Original: last,
			})
			last = parsed
			hasLast = true
		}
	}

	// 循环结束后，如果还有未配对的最后一行，作为单独原文行加入
	if hasLast {
		l.pairs = append(l.pairs, lyricPair{
			Original: last,
		})
	}
}

func parseLine(line string) (lyricLine, error) {
	var ll lyricLine
	pattern := `\[(\d{2}):(\d{2})\.(\d{2,3})\]([^[]*)`
	regex := regexp.MustCompile(pattern)
	matches := regex.FindAllStringSubmatch(line, -1)
	if len(matches) < 1 {
		return ll, fmt.Errorf("no match found")
	} else if len(matches) == 1 {
		w, err := parseWord(matches[0])
		if err != nil {
			return ll, err
		}
		ll.Time = w.Time
		ll.Words = []word{w}
		ll.Text = w.Text
		return ll, nil
	}

	text := ""
	for i, match := range matches {
		w, err := parseWord(match)
		if err != nil {
			return ll, err
		}
		if strings.TrimSpace(w.Text) == "" {
			continue
		}
		if i == 0 {
			ll.Time = w.Time
		}
		ll.Words = append(ll.Words, w)
		text += w.Text
	}
	ll.Text = text
	return ll, nil
}

func parseWord(wordInfo []string) (word, error) {
	min, err := time.ParseDuration(wordInfo[1] + "m")
	if err != nil {
		return word{}, err
	}
	sec, err := time.ParseDuration(wordInfo[2] + "s")
	if err != nil {
		return word{}, err
	}
	// 处理小数部分：如果是两位（如 "34"），表示 340ms；如果三位则直接是 ms
	msStr := wordInfo[3]
	msInt, err := strconv.Atoi(msStr)
	if err != nil {
		return word{}, err
	}
	if len(msStr) == 2 {
		msInt *= 10
	}
	ms := time.Duration(msInt) * time.Millisecond

	return word{
		Time: min + sec + ms,
		Text: wordInfo[4],
	}, nil
}

func (l *lyrics) print(wg *sync.WaitGroup, player *Player) {
	defer wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-player.done:
			return
		case <-ticker.C:
			currentTime := player.getCurrentTime()
			l.printCurrentLyric(currentTime)
		}
	}
}

func (l *lyrics) getCurrentLyric(currentTime time.Duration) (int, lyricPair) {
	n := len(l.pairs)
	if n == 0 {
		return -1, lyricPair{}
	}
	// 在第一行之前
	if currentTime < l.pairs[0].Original.Time {
		return -1, lyricPair{}
	}
	// 在某行与下一行之间，返回当前行索引
	for i := 0; i < n-1; i++ {
		if currentTime >= l.pairs[i].Original.Time && currentTime < l.pairs[i+1].Original.Time {
			return i, l.pairs[i]
		}
	}
	// 在最后一行及之后
	return n - 1, l.pairs[n-1]
}

func (l *lyrics) printCurrentLyric(currentTime time.Duration) {
	var lastOriginalLine lyricLine
	var currentOriginalLine lyricLine
	var nextOriginalLine lyricLine
	var lastTranslatedLine lyricLine
	var currentTranslatedLine lyricLine
	var nextTranslatedLine lyricLine

	lIndex, currentLine := l.getCurrentLyric(currentTime)

	// 当前行（可能为零值，如果 lIndex == -1）
	if lIndex >= 0 {
		currentOriginalLine = currentLine.Original
		currentTranslatedLine = l.pairs[lIndex].Translated
	}

	if lIndex-1 >= 0 {
		lastOriginalLine = l.pairs[lIndex-1].Original
		lastTranslatedLine = l.pairs[lIndex-1].Translated
	}
	if lIndex+1 < len(l.pairs) {
		nextOriginalLine = l.pairs[lIndex+1].Original
		nextTranslatedLine = l.pairs[lIndex+1].Translated
	}

	// 使用当前原文计算当前字索引（如果没有当前原文，getCurrentWord 会返回 -1）
	wIndex, _ := getCurrentWord(currentOriginalLine, currentTime)

	printMu.Lock()
	defer printMu.Unlock()

	fmt.Print("\033[2;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center("\033[40;30m======防闪烁======\033[0m"))

	fmt.Print("\033[3;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center("\033[40;30m======防闪烁======\033[0m"))

	fmt.Print("\033[4;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center(lastOriginalLine.Text))

	fmt.Print("\033[5;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center(lastTranslatedLine.Text))

	fmt.Print("\033[7;1H")
	fmt.Print("\033[2K")
	fmt.Print(utils.Center("\x1b[34m➣ " + l.getWordText(currentOriginalLine, wIndex)))

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

func getCurrentWord(lyricLine lyricLine, currentTime time.Duration) (int, word) {
	if len(lyricLine.Words) == 0 {
		return -1, word{}
	}
	for i, w := range lyricLine.Words {
		if currentTime < w.Time {
			// 如果还没到第一个字，返回 -1 表示没有已播放的字
			if i == 0 {
				return -1, word{}
			}
			return i - 1, lyricLine.Words[i-1]
		}
	}
	// 已经超过最后一个字，返回最后一个字
	last := len(lyricLine.Words) - 1
	return last, lyricLine.Words[last]
}

func (l *lyrics) getWordText(line lyricLine, index int) string {
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
