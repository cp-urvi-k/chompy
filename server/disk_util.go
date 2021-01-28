package server

import (
	"io/ioutil"
	"sort"
	"strings"
)

type ordering int

const (
	createdAsc ordering = iota + 1
	createdDesc
)

const toMiB = 1024 * 1024

//TODO: tests
func getVideoFiles(path string, order ordering) ([]videoFile, error) {
	var vids []videoFile

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return vids, err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") || file.IsDir() {
			continue
		}
		vids = append(vids, videoFile{
			Filename: file.Name(),
			Created:  file.ModTime(),
			Size:     file.Size() / toMiB,
		})
	}

	switch order {
	case createdAsc:
		sort.Slice(vids, func(i, j int) bool { return vids[i].Created.After(vids[j].Created) })
	case createdDesc:
		sort.Slice(vids, func(i, j int) bool { return vids[i].Created.Before(vids[j].Created) })
	}

	return vids, nil
}

//TODO: tests

func deleteVideoFiles(vids []videoFile) error {
	return nil
}
