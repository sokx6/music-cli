package utils

import (
	"os"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

func Center(text string) string {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // 默认宽度
	}
	textWidth := runewidth.StringWidth(text)
	if textWidth >= width {
		return text // 如果文本宽度大于等于终端宽度，直接返回原文本
	}
	padding := (width - textWidth) / 2
	return runewidth.FillLeft(text, textWidth+padding)
}
