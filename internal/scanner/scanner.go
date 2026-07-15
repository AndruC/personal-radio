package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

type Track struct {
	Path string
	Name string
}

var supportedExts = map[string]bool{
	".mp3": true,
	".ogg": true,
}

func Scan(root string) ([]Track, error) {
	var tracks []Track
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if supportedExts[ext] {
			tracks = append(tracks, Track{
				Path: path,
				Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tracks, nil
}
