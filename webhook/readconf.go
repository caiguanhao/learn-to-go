package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Conf struct {
	FilePaths []string
	Configs   [][]string
}

func (conf *Conf) SetFilePaths(paths ...string) {
	(*conf).FilePaths = paths
}

func (conf *Conf) Read() (string, bool) {
	read := func(reader io.Reader) bool {
		scanner := bufio.NewScanner(reader)
		if err := scanner.Err(); err != nil {
			return false
		}
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(text, "#") {
				continue
			}
			comp := strings.SplitN(text, " ", 2)
			if len(comp) != 2 {
				continue
			}
			comp[1] = strings.TrimSpace(comp[1])
			(*conf).Configs = append((*conf).Configs, comp)
		}
		return true
	}
	(*conf).Configs = nil
	for _, fpath := range (*conf).FilePaths {
		fullpath, err := filepath.Abs(fpath)
		if err != nil {
			continue
		}
		file, err := os.Open(fullpath)
		defer file.Close()
		if err != nil {
			continue
		}
		if !read(file) {
			continue
		}
		return fullpath, true
	}
	if read(os.Stdin) {
		return "", true
	}
	return "", false
}

func (conf *Conf) Get(key string) (string, bool) {
	for _, item := range (*conf).Configs {
		if item[0] == key {
			return item[1], true
		}
	}
	return "", false
}

func (conf *Conf) GetByRepoEvent(getWhat, repo, event string) (string, bool) {
	phase := 0
	for _, item := range (*conf).Configs {
		if item[0] == "repository" && item[1] == repo {
			if phase == 0 {
				phase = 1
				continue
			} else {
				break
			}
		}
		if phase == 1 && item[0] == "event" && item[1] == event {
			phase = 2
			continue
		}
		if phase == 2 && item[0] == getWhat {
			return item[1], true
		}
	}
	return "", false
}
