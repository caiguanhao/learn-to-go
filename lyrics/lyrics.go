package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path"
	"strings"
	"time"
)

const (
	USAGE = `Usage: %s [OPTION] [of SONG NAME [by ARTIST]]

  -h, --help         Show this content and exit
  -P, --no-pager     Don't pipe output into a pager
  -C, --no-cache     Don't read/write lyrics from/to cache
`
)

var (
	cmd    *exec.Cmd
	reader *io.PipeReader
	writer *io.PipeWriter

	hasStartupQuery bool

	noPager        bool
	pager          bool
	isPagerRunning bool

	noCache bool
	cache   bool

	currentTrack *Track

	failedOnce bool

	userHomeDir    string
	lyricsCacheDir string
)

func getCurrentTrack() bool {
	changed, err := currentTrack.ITunes.GetCurrentTrack()

	if err != nil {
		if !failedOnce {
			errorln("Couldn't get information from iTunes.")
			errorln("Are you sure you have opened iTunes and it is playing some music?")
			failedOnce = true
		}
		return false
	}

	return changed
}

func errorln(a ...interface{}) {
	if writer == nil {
		fmt.Fprintf(os.Stderr, "Error: ")
		fmt.Fprintln(os.Stderr, a...)
	} else {
		fmt.Fprintf(writer, "Error: ")
		fmt.Fprintln(writer, a...)
	}
}

func init() {
	currentUser, _ := user.Current()
	userHomeDir = currentUser.HomeDir
	lyricsCacheDir = path.Join(userHomeDir, ".lyrics")

	flag.BoolVar(&noPager, "no-pager", false, "")
	flag.BoolVar(&noPager, "P", false, "")
	flag.BoolVar(&noCache, "no-cache", false, "")
	flag.BoolVar(&noCache, "C", false, "")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, USAGE, path.Base(os.Args[0]))
	}
	flag.Parse()
	rest := flag.NArg()
	if rest > 0 {
		start := 0
		if strings.EqualFold(flag.Arg(0), "of") {
			start++
		}
		var name, artist []string
		var byed bool
		for index, arg := range flag.Args()[start:] {
			arg = strings.TrimSpace(arg)
			if len(arg) < 1 {
				continue
			}
			if byed {
				artist = append(artist, arg)
			} else if strings.EqualFold(arg, "by") ||
				(index > 0 && strings.EqualFold(arg, "of")) {
				byed = true
			} else {
				name = append(name, arg)
			}
		}
		if len(name) == 0 {
			errorln("You need to specify the name of the song.")
			os.Exit(1)
		}
		currentTrack = NewTrack(strings.Join(name, " "), strings.Join(artist, " "))
		hasStartupQuery = true
	}
	pager = !noPager
	cache = !noCache

	if currentTrack == nil {
		currentTrack = NewTrack("", "")
	}
}

func trapCtrlC() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			fmt.Print(" You can press 'q' to exit. ")
		}
	}()
}

func findLyrics() {
	var filename string
	var lyrics []byte
	var err error

	fn, _, cacheable := (*currentTrack).AZLyrics.BuildFileName()
	filename = path.Join(lyricsCacheDir, fn[0])
	if cache && cacheable {
		lyrics, err = ioutil.ReadFile(filename)
	}
	if err != nil || len(lyrics) == 0 {
		for _, provider := range []Provider{
			(*currentTrack).AZLyrics,
			(*currentTrack).AZLyricDBCN,
		} {
			lyrics = provider.GetLyrics()
			if len(lyrics) > 0 {
				break
			}
		}

		if cache && cacheable && len(lyrics) > 0 && filename != "" {
			err = os.MkdirAll(path.Dir(filename), 0755)
			if err == nil {
				ioutil.WriteFile(filename, lyrics, 0644)
			}
		}
	}

	if len(lyrics) > 0 {
		if writer == nil {
			fmt.Fprintf(os.Stdout, "%s\n", lyrics)
		} else {
			fmt.Fprintf(writer, "%s\n", lyrics)
		}
	} else {
		errorln(fmt.Sprintf("No lyrics found for %s - %s.",
			(*currentTrack).Name, (*currentTrack).Artist))
	}
}

func runPager() {
	for {
		reader, writer = io.Pipe()
		cmd = exec.Command("less")
		cmd.Stdin = reader
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		isPagerRunning = true
		cmd.Run()
		if cmd.ProcessState.Success() {
			break
		}
	}
}

func main() {

	if pager {

		var started bool = false

		go func() {
			for {
				if hasStartupQuery || getCurrentTrack() {
					if started {
						cmd.Process.Kill()
						isPagerRunning = false
					}

					for !isPagerRunning {
						time.Sleep(100 * time.Millisecond)
					}

					findLyrics()

					started = true
					writer.Close()
				}

				if hasStartupQuery {
					break
				}

				time.Sleep(500 * time.Millisecond)
			}
		}()

		trapCtrlC()

		runPager()

	} else {

		if hasStartupQuery || getCurrentTrack() {
			findLyrics()
		}

	}

}
