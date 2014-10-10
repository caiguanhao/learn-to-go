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

func (conf *Conf) GetCommandByEvent(event string) (string, bool) {
	getCommand := false
	for _, item := range (*conf).Configs {
		if item[0] == "event" && item[1] == event {
			getCommand = true
			continue
		}
		if getCommand && item[0] == "command" {
			return item[1], true
		}
	}
	return "", false
}
