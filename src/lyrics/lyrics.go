package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"strings"
	"time"
)

type Track struct {
	Name   string
	Artist string
}

func (track Track) Query() string {
	if &track == nil {
		return ""
	}
	name := track.Name
	re := regexp.MustCompile("(\\[|\\().+?(\\]|\\))") // (.*) [.*]
	name = re.ReplaceAllString(name, "")
	re = regexp.MustCompile("(?i)f[uc*]{2}k") // fuck
	name = re.ReplaceAllString(name, "")
	artist := track.Artist
	artist = strings.Replace(artist, "!", "i", -1) // P!nk
	return name + " " + artist
}

var currentTrack *Track

var (
	osascript = `
	if application "iTunes" is running then
		tell application "iTunes"
			(get name of current track) & "\n" & (get artist of current track)
		end tell
	end if`
)

var failedOnce bool

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
	if currentTrack == nil || (*currentTrack).Name != info[0] || (*currentTrack).Artist != info[1] {
		currentTrack = &Track{
			Name:   info[0],
			Artist: info[1],
		}
		return true
	}
	return false
}

func findOnAZLyrics(__query__ string) []string {
	results := []string{}
	if __query__ == "" {
		return results
	}
	query := url.Values{}
	query.Add("q", __query__)
	URL := url.URL{
		Scheme:   "http",
		Host:     "search.azlyrics.com",
		Path:     "search.php",
		RawQuery: query.Encode(),
	}
	doc, err := goquery.NewDocument(URL.String())
	if err != nil {
		errorln("Failed to get lyrics.")
		return results
	}
	doc.Find("a").Each(func(i int, anchor *goquery.Selection) {
		href, _ := anchor.Attr("href")
		if strings.HasPrefix(href, "http://www.azlyrics.com/lyrics/") {
			results = append(results, href)
		}
	})
	return results
}

func findOnAZLyricsByTrack(track *Track) []string {
	return findOnAZLyrics((*track).Query())
}

func getLyrics(lyricsURL string) string {
	songPage, err := goquery.NewDocument(lyricsURL)
	if err != nil {
		errorln("Failed to get lyrics.")
		return ""
	}
	song := strings.TrimSpace(songPage.Find("#main > b").First().Text())
	artist := songPage.Find("#main > h2").First().Text()
	artist = strings.Replace(artist, "LYRICS", "", -1)
	artist = strings.TrimSpace(artist)
	lyrics := strings.TrimSpace(songPage.Find("#main > div[style]").Text())
	return fmt.Sprintf("%s by %s\n\n%s", song, artist, lyrics)
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

var (
	cmd    *exec.Cmd
	reader *io.PipeReader
	writer *io.PipeWriter

	hasStartupQuery bool
	startupQuery    string

	noPager bool
	pager   bool
)

func init() {
	flag.BoolVar(&noPager, "no-pager", false, "Don't pipe output into a pager")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION] [of [SONG NAME [by [ARTIST]]]]\n\n",
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
		if flag.Arg(0) == "of" {
			start++
		}
		args := flag.Args()
		startupQuery = strings.Join(args[start:len(args)], " ")
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
	var results []string

	if hasStartupQuery {
		results = findOnAZLyrics(startupQuery)
	} else {
		results = findOnAZLyricsByTrack(currentTrack)
	}

	if len(results) == 0 {
		if hasStartupQuery {
			errorln(fmt.Sprintf("No lyrics found for %s.",
				startupQuery))
		} else {
			errorln(fmt.Sprintf("No lyrics found for %s - %s.",
				(*currentTrack).Name, (*currentTrack).Artist))
		}
	} else {
		lyrics := getLyrics(results[0])
		if writer == nil {
			fmt.Fprintln(os.Stdout, lyrics)
		} else {
			fmt.Fprintln(writer, lyrics)
		}
	}
}

func runPager() {
	for {
		reader, writer = io.Pipe()
		cmd = exec.Command("less")
		cmd.Stdin = reader
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
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
					}

					findLyrics()

					started = true
					writer.Close()
				}

				if hasStartupQuery {
					break
				}

				time.Sleep(1 * time.Second)
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
