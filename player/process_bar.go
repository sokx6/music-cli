package player

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

type progressBar struct {
	totalTime   time.Duration
	currentTime time.Duration
}

func newProgressBar(total time.Duration) *progressBar {
	return &progressBar{
		totalTime: total,
	}
}

func (pb *progressBar) getCurrentBar() string {
	fd := int(os.Stdout.Fd())
	width, _, err := term.GetSize(fd)
	if err != nil {
		width = 50 //默认值50
	}
	percentage := float64(pb.currentTime) / float64(pb.totalTime) * 100
	if percentage > 100 {
		percentage = 100
	}

	// 减去一些空间用于显示时间，防止进度条过长导致换行
	currentBarLength := width - 17
	if currentBarLength <= 0 { // 防止窗口太小时长度为负
		currentBarLength = 1
	}

	// 计算当前时间的分钟和秒数
	currentMinute := int(pb.currentTime.Minutes())
	currentSecond := int(pb.currentTime.Seconds()) % 60

	// 计算总时间的分钟和秒数
	totalMinute := int(pb.totalTime.Minutes())
	totalSecond := int(pb.totalTime.Seconds()) % 60

	// 构建进度条字符串
	bar := "  "
	bar += "\x1b[0m"                                               // 重置所有颜色
	bar += fmt.Sprintf("%02d:%02d ", currentMinute, currentSecond) // 显示当前时间

	// 计算已播放的长度
	filledLength := int(percentage / 100 * float64(currentBarLength))

	for i := 0; i < currentBarLength-1; i++ {
		if i < filledLength {
			bar += "\x1b[34m█" // 蓝色已播放部分
		} else {
			bar += "\x1b[30;1m█" // 深灰色未播放部分
		}
	}

	bar += fmt.Sprintf(" \x1b[0m%02d:%02d", totalMinute, totalSecond) // 重置颜色并显示总时间
	return bar
}

func (pb *progressBar) printBar(wg *sync.WaitGroup, player *Player) {
	defer wg.Done()

	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	for {
		select {
		case <-player.done:
			return
		case <-ticker.C:
			pb.currentTime = player.getCurrentTime()
			printMu.Lock()
			fmt.Printf("\033[1;1f")
			fmt.Printf("\033[K %s", pb.getCurrentBar())
			printMu.Unlock()
		}
	}
}
