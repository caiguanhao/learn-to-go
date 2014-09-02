package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Track struct {
	Name   string
	Artist string
}

func (track Track) Query() string {
	name := track.Name
	artist := track.Artist
	artist = strings.Replace(artist, "!", "i", -1)
	return name + " " + artist
}

func getCurrentTrack() Track {
	output, err := exec.Command("osascript", "-e",
		"tell application \"iTunes\" to (get name of current track) & \"\n\""+
			" & (get artist of current track)").Output()
	if err != nil {
		log.Fatal("iTunes: ", err)
	}
	info := strings.Split(strings.TrimSpace(string(output)), "\n")
	return Track{
		Name:   info[0],
		Artist: info[1],
	}
}

func findOnAZLyrics(track Track) []string {
	query := url.Values{}
	query.Add("q", track.Query())
	URL := url.URL{
		Scheme:   "http",
		Host:     "search.azlyrics.com",
		Path:     "search.php",
		RawQuery: query.Encode(),
	}
	doc, err := goquery.NewDocument(URL.String())
	if err != nil {
		log.Fatal(err)
	}
	results := []string{}
	doc.Find("a").Each(func(i int, anchor *goquery.Selection) {
		href, _ := anchor.Attr("href")
		if strings.HasPrefix(href, "http://www.azlyrics.com/lyrics/") {
			results = append(results, href)
		}
	})
	return results
}

func getLyrics(lyricsURL string) string {
	songPage, err := goquery.NewDocument(lyricsURL)
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSpace(songPage.Find("#main > div[style]").Text())
}

func main() {
	currentTrack := getCurrentTrack()
	results := findOnAZLyrics(currentTrack)

	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "No lyrics found for %s - %s.\n",
			currentTrack.Name, currentTrack.Artist)
		os.Exit(1)
	}

	reader, writer := io.Pipe()

	cmd := exec.Command("less")
	cmd.Stdin = reader
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	c := make(chan int)

	go func() {
		defer close(c)
		cmd.Run()
	}()

	lyrics := getLyrics(results[0])
	fmt.Fprintln(writer, lyrics)

	writer.Close()
	<-c
}
