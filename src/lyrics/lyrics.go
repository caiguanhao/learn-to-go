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

var (
	cmd    *exec.Cmd
	reader *io.PipeReader
	writer *io.PipeWriter

	hasStartupQuery bool
	startupQuery    string

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
)

type (
	Track struct {
		Name   string
		Artist string
	}

	Result struct {
		URL       string
		TrackName string
	}
)

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

func (trackA Track) Equal(trackB Track) bool {
	return trackA.Name == trackB.Name && trackA.Artist == trackB.Artist
}

func (trackA Track) NotEqual(trackB Track) bool {
	return !trackA.Equal(trackB)
}

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

	newTrack := &Track{
		Name:   info[0],
		Artist: info[1],
	}

	if currentTrack == nil || (*currentTrack).NotEqual(*newTrack) {
		currentTrack = newTrack
		return true
	}

	newTrack = nil

	return false
}

func findOnAZLyrics(__query__ string) []Result {
	results := []Result{}
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
			results = append(results, Result{
				URL:       href,
				TrackName: anchor.Text(),
			})
		}
	})
	return results
}

func findOnAZLyricsByTrack(track *Track) []Result {
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

func collapse(input string) string {
	return strings.Replace(strings.ToLower(input), " ", "", -1)
}

func filterPossibleResultByTrack(results *[]Result, track *Track) string {
	if len(*results) < 1 {
		return ""
	}
	var index int = 0
	for i, result := range *results {
		if collapse(result.TrackName) == collapse((*track).Name) {
			index = i
			break
		}
	}
	return (*results)[index].URL
}

func score(long, short string) float64 {
	return 1 - float64(len(strings.Replace(long, short, "", -1)))/float64(len(long))
}

// you can test with `lyrics --no-pager Mean Pink`
func filterPossibleResultByQuery(results *[]Result, query string) string {
	if len(*results) < 1 {
		return ""
	}
	var index int = 0
	var max float64
	parts := strings.Split(query, " ")

	for i, result := range *results {
		for _, part := range parts {
			per := score(collapse(result.TrackName), collapse(part))
			if per > max {
				max = per
				index = i
			}
		}
	}

	return (*results)[index].URL
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
	var results []Result
	var link string

	if hasStartupQuery {
		results = findOnAZLyrics(startupQuery)
		link = filterPossibleResultByQuery(&results, startupQuery)
	} else {
		results = findOnAZLyricsByTrack(currentTrack)
		link = filterPossibleResultByTrack(&results, currentTrack)
	}

	if len(results) == 0 {
		if hasStartupQuery {
			errorln(fmt.Sprintf("No lyrics found for %s.",
				startupQuery))
		} else {
			errorln(fmt.Sprintf("No lyrics found for %s - %s.",
				(*currentTrack).Name, (*currentTrack).Artist))
		}
	} else if link != "" {
		lyrics := getLyrics(link)
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
