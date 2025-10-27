package utils

import (
	"os"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

func Center(text string) string {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // 默认宽度
	}
	plainText := stripansi.Strip(text)
	textWidth := runewidth.StringWidth(plainText)
	if textWidth >= width {
		return text // 如果文本宽度大于等于终端宽度，直接返回原文本
	}
	padding := (width - textWidth) / 2
	return strings.Repeat(" ", padding) + text
}
