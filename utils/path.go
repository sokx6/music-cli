package utils

import (
	"io/fs"
	"path/filepath"
)

func GetPaths(root string) ([]string, error) {
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
