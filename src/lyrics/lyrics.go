package main

import (
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
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

func getCurrentTrack() bool {
	output, err := exec.Command("osascript", "-e",
		"tell application \"iTunes\" to (get name of current track) & \"\n\""+
			" & (get artist of current track)").Output()
	if err != nil {
		errorln("Couldn't get information from iTunes.")
		errorln("Are you sure you have opened iTunes and it is playing some music?")
		return false
	}
	info := strings.Split(strings.TrimSpace(string(output)), "\n")
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
	fmt.Fprintf(writer, "Error: ")
	fmt.Fprintln(writer, a...)
}

var cmd *exec.Cmd
var reader *io.PipeReader
var writer *io.PipeWriter

var (
	hasStartupQuery bool
	startupQuery    string
)

func init() {
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
}

func main() {
	var started bool = false

	go func() {
		for {
			if hasStartupQuery || getCurrentTrack() {
				if started {
					cmd.Process.Kill()
				}

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
					fmt.Fprintln(writer, lyrics)
				}
				started = true
				writer.Close()
			}

			if hasStartupQuery {
				break
			}

			time.Sleep(1 * time.Second)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			fmt.Print(" You can press 'q' to exit. ")
		}
	}()

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
