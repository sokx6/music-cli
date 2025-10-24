package player

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

type ProgressBar struct {
	totalTime   time.Duration
	currentTime time.Duration
	barLength   int // 这个变量现在主要用来存储一个默认值，或者可以移除
}

// NewProgressBar 构造并返回一个 ProgressBar 的指针
// 注意：这里不再需要 barLength 参数，因为我们每次都会动态获取
func NewProgressBar(total time.Duration) *ProgressBar {
	return &ProgressBar{
		totalTime: total,
	}
}

// PrintBar 打印进度条，并动态适应终端大小变化
func (pb *ProgressBar) PrintBar() {
	// 隐藏光标
	fmt.Print("\x1b[?25l")
	// 确保在函数退出时恢复光标显示
	defer fmt.Print("\x1b[?25h")

	fd := int(os.Stdout.Fd())

	for ; pb.currentTime <= pb.totalTime; pb.currentTime += time.Millisecond * 50 {
		// *** 核心改动：在每次循环开始时，都获取最新的终端大小 ***
		width, _, err := term.GetSize(fd)
		if err != nil {
			// 如果获取失败，可以使用一个安全的默认值，或者直接跳过此次更新
			// 这里我们使用一个默认值 50，以防止程序崩溃
			width = 65
		}
		// 动态计算进度条的长度
		// 减去一些空间用于显示时间，防止进度条过长导致换行
		currentBarLength := width - 15
		if currentBarLength <= 0 { // 防止窗口太小时长度为负
			currentBarLength = 1
		}

		// 计算当前进度百分比
		percentage := float64(pb.currentTime) / float64(pb.totalTime) * 100
		if percentage > 100 {
			percentage = 100
		}

		// 计算当前时间的分钟和秒数
		currentMinute := int(pb.currentTime.Minutes())
		currentSecond := int(pb.currentTime.Seconds()) % 60

		// 计算总时间的分钟和秒数
		totalMinute := int(pb.totalTime.Minutes())
		totalSecond := int(pb.totalTime.Seconds()) % 60

		// 构建进度条字符串
		// 使用 \r 将光标移动到行首，然后重绘整行
		bar := "\x1b[0m" // 重置所有颜色
		bar += fmt.Sprintf("%02d:%02d ", currentMinute, currentSecond)

		// *** 使用动态计算出的 currentBarLength ***
		filledLength := int(percentage / 100 * float64(currentBarLength))
		for i := 0; i < currentBarLength; i++ {
			if i < filledLength {
				bar += "\x1b[34m█" // 蓝色已填充部分
			} else {
				bar += "\x1b[30;1m█" // 深灰色未填充部分
			}
		}
		bar += fmt.Sprintf(" \x1b[0m%02d:%02d", totalMinute, totalSecond) // 重置颜色并显示总时间

		fmt.Print("\r")      // 使用 \r 后直接跟上完整的字符串
		fmt.Print("\x1b[2K") // 清除行内容
		fmt.Print(bar)
		_ = os.Stdout.Sync() // 确保输出被立即刷新

		time.Sleep(time.Millisecond * 50)
	}

	// 循环结束后，将光标移动到新的一行
	fmt.Println()
}
