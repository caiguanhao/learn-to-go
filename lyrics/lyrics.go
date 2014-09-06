package main

import (
	"bytes"
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

var (
	cmd            *exec.Cmd
	reader         *io.PipeReader
	writer         *io.PipeWriter
	isPagerRunning bool

	hasStartupQuery bool

	noPager bool
	pager   bool

	currentTrack *Track

	osascript = `
	if application "iTunes" is running then
		tell application "iTunes"
			(get name of current track) & "\n" & (get artist of current track)
		end tell
	end if`

	failedOnce bool

	userHomeDir    string
	lyricsCacheDir string
)

func getCurrentTrack() bool {
	var output bytes.Buffer
	var failed bool
	var out string

	cmd := exec.Command("osascript")
	cmd.Stdin = strings.NewReader(osascript)
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		failed = true
	} else {
		out = strings.TrimSpace(output.String())
		if out == "" {
			failed = true
		}
	}

	if failed {
		if !failedOnce {
			errorln("Couldn't get information from iTunes.")
			errorln("Are you sure you have opened iTunes and it is playing some music?")
			failedOnce = true
		}
		return false
	}

	info := strings.Split(out, "\n")

	newTrack := NewTrack(info[0], info[1])

	if currentTrack == nil || (*currentTrack).NotEqual(*newTrack) {
		currentTrack = newTrack
		return true
	}

	newTrack = nil

	return false
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

	flag.BoolVar(&noPager, "no-pager", false, "Don't pipe output into a pager")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION] [of SONG NAME by ARTIST]\n\n",
			path.Base(os.Args[0]))
		flag.VisitAll(func(flag *flag.Flag) {
			switch flag.DefValue {
			case "true", "false", "":
				fmt.Fprintf(os.Stderr, "  --%s  %s\n", flag.Name, flag.Usage)
			default:
				fmt.Fprintf(os.Stderr, "  --%s  %s, default is %s\n",
					flag.Name, flag.Usage, flag.DefValue)
			}
		})
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
		for _, arg := range flag.Args()[start:] {
			arg = strings.TrimSpace(arg)
			if len(arg) < 1 {
				continue
			}
			if byed {
				artist = append(artist, arg)
			} else if strings.EqualFold(arg, "by") {
				byed = true
			} else {
				name = append(name, arg)
			}
		}
		if len(name) == 0 {
			errorln("You need to specify the name of the song.")
			os.Exit(1)
		}
		if len(artist) == 0 {
			errorln("You need to specify the artist of the song.")
			os.Exit(1)
		}
		currentTrack = NewTrack(strings.Join(name, " "), strings.Join(artist, " "))
		hasStartupQuery = true
	}
	pager = !noPager
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

	filename = path.Join(lyricsCacheDir, (*currentTrack).AZLyrics.BuildFileName()[0])
	lyrics, err = ioutil.ReadFile(filename)
	if err != nil || len(lyrics) == 0 {
		lyrics = (*currentTrack).AZLyrics.GetLyrics()

		if len(lyrics) > 0 && filename != "" {
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
