package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/term"
)

func WalkDir(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // 如果访问路径出错，则停止遍历
		}
		if !d.IsDir() && (filepath.Ext(d.Name()) == ".mp3" || filepath.Ext(d.Name()) == ".flac" || filepath.Ext(d.Name()) == ".wav") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func ListDir(root string) ([]string, []string, error) {
	var files []string
	var dirs []string

	info, err := os.Lstat(root)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		files = append(files, root)
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, filepath.Join(root, entry.Name()))
			} else if filepath.Ext(entry.Name()) == ".mp3" || filepath.Ext(entry.Name()) == ".flac" || filepath.Ext(entry.Name()) == ".wav" {
				files = append(files, filepath.Join(root, entry.Name()))
			}

		}
	}
	return files, dirs, nil
}

func PrintPathInfo(root string, page int) {
	if page < 1 {
		page = 1
	}
	_, height, _ := term.GetSize(int(os.Stdout.Fd()))
	fmt.Printf("当前路径: %s\n", root)
	fmt.Printf("页码: %d\n", page)
	pageSize := height - 6
	if pageSize <= 0 {
		pageSize = 1
	}
	fmt.Printf("可显示行数: %d\n", pageSize)

	files, dirs, err := ListDir(root)
	if err != nil {
		fmt.Println("错误:", err)
		return
	}

	total := len(files) + len(dirs)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		fmt.Println("没有更多内容。")
		return
	}

	// 打印歌曲（如果在当前页范围内）
	fmt.Println("歌曲：")
	fileStart := start
	if fileStart < 0 {
		fileStart = 0
	}
	fileEnd := end
	if fileEnd > len(files) {
		fileEnd = len(files)
	}
	for i := fileStart; i < fileEnd; i++ {
		fmt.Printf(" %d. %s\n", i+1, filepath.Base(files[i]))
	}

	// 打印目录（如果在当前页范围内）
	fmt.Println("目录：")
	dirStart := start - len(files)
	if dirStart < 0 {
		dirStart = 0
	}
	dirEnd := end - len(files)
	if dirEnd < 0 {
		dirEnd = 0
	}
	if dirStart < len(dirs) {
		if dirEnd > len(dirs) {
			dirEnd = len(dirs)
		}
		for j := dirStart; j < dirEnd; j++ {
			fmt.Printf(" %d. %s\n", len(files)+j+1, filepath.Base(dirs[j]))
		}
	}

	if end < total {
		fmt.Println("...（更多内容省略）")
	}
}
