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

type Pager struct {
	Cmd        *exec.Cmd
	Reader     *io.PipeReader
	Writer     *io.PipeWriter
	Running    bool
	FailedOnce bool
}

type Options struct {
	HasStartupQuery bool
	NoPager         bool
	Pager           bool
	NoCache         bool
	Cache           bool
}

var (
	options Options
	pager   Pager

	currentTrack *Track
)

func getCurrentTrack() bool {
	changed, err := currentTrack.ITunes.GetCurrentTrack()

	if err != nil {
		if !pager.FailedOnce {
			errorln("Couldn't get information from iTunes.")
			errorln("Are you sure you have opened iTunes and it is playing some music?")
			pager.FailedOnce = true
		}
		return false
	}

	return changed
}

func errorln(a ...interface{}) {
	if pager.Writer == nil {
		fmt.Fprintf(os.Stderr, "Error: ")
		fmt.Fprintln(os.Stderr, a...)
	} else {
		fmt.Fprintf(pager.Writer, "Error: ")
		fmt.Fprintln(pager.Writer, a...)
	}
}

func init() {
	flag.BoolVar(&options.NoPager, "no-pager", false, "")
	flag.BoolVar(&options.NoPager, "P", false, "")
	flag.BoolVar(&options.NoCache, "no-cache", false, "")
	flag.BoolVar(&options.NoCache, "C", false, "")
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
		options.HasStartupQuery = true
	}
	options.Pager = !options.NoPager
	options.Cache = !options.NoCache

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

func findLyrics(lyricsCacheDir *string) {
	var filename string
	var lyrics []byte
	var err error

	fn, _, cacheable := (*currentTrack).AZLyrics.BuildFileName()
	filename = path.Join(*lyricsCacheDir, fn[0])
	if options.Cache && cacheable {
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

		if options.Cache && cacheable && len(lyrics) > 0 && filename != "" {
			err = os.MkdirAll(path.Dir(filename), 0755)
			if err == nil {
				ioutil.WriteFile(filename, lyrics, 0644)
			}
		}
	}

	if len(lyrics) > 0 {
		if pager.Writer == nil {
			fmt.Fprintf(os.Stdout, "%s\n", lyrics)
		} else {
			fmt.Fprintf(pager.Writer, "%s\n", lyrics)
		}
	} else {
		errorln(fmt.Sprintf("No lyrics found for %s - %s.",
			(*currentTrack).Name, (*currentTrack).Artist))
	}
}

func runPager() {
	for {
		pager.Reader, pager.Writer = io.Pipe()
		pager.Cmd = exec.Command("less")
		pager.Cmd.Stdin = pager.Reader
		pager.Cmd.Stdout = os.Stdout
		pager.Cmd.Stderr = os.Stderr
		pager.Running = true
		pager.Cmd.Run()
		if pager.Cmd.ProcessState.Success() {
			break
		}
	}
}

func main() {

	currentUser, err := user.Current()
	if err != nil {
		errorln("Unable to get current user.")
		os.Exit(1)
	}
	userHomeDir := currentUser.HomeDir
	lyricsCacheDir := path.Join(userHomeDir, ".lyrics")

	if options.Pager {

		var started bool = false

		go func() {
			for {
				if options.HasStartupQuery || getCurrentTrack() {
					if started {
						pager.Cmd.Process.Kill()
						pager.Running = false
					}

					for !pager.Running {
						time.Sleep(100 * time.Millisecond)
					}

					findLyrics(&lyricsCacheDir)

					started = true
					pager.Writer.Close()
				}

				if options.HasStartupQuery {
					break
				}

				time.Sleep(500 * time.Millisecond)
			}
		}()

		trapCtrlC()

		runPager()

	} else {

		if options.HasStartupQuery || getCurrentTrack() {
			findLyrics(&lyricsCacheDir)
		}

	}

}
