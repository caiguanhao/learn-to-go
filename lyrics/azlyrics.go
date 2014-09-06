package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
)

const (
	AZLYRICS = "http://www.azlyrics.com/lyrics/"
)

func az0(input string) string {
	var re *regexp.Regexp

	input = strings.ToLower(input)
	input = strings.Replace(input, "p!nk", "pink", -1)

	re = regexp.MustCompile("(?i)f[uc*]{2}k") // fuck
	input = re.ReplaceAllString(input, "")

	re = regexp.MustCompile("\\[.+?\\]") // [.*]
	input = re.ReplaceAllString(input, "")

	return input
}

func az1(input string) string {
	var re *regexp.Regexp

	input = az0(input)

	re = regexp.MustCompile("[^\\w]")
	input = re.ReplaceAllString(input, "")

	return input
}

func az2(input string) string {
	var re *regexp.Regexp

	input = az0(input)

	re = regexp.MustCompile("\\(.+?\\)") // (.*)
	input = re.ReplaceAllString(input, "")

	re = regexp.MustCompile("[^\\w]")
	input = re.ReplaceAllString(input, "")

	return input
}

func BuildFileNameForTrack(track Track) []string {
	artist := az2(track.Artist)
	u1 := az1(track.Name)
	u2 := az2(track.Name)
	ret := []string{
		fmt.Sprintf("%s/%s", artist, u1),
	}
	if u1 != u2 {
		ret = append(ret, fmt.Sprintf("%s/%s", artist, u2))
	}
	return ret
}

func GetLyricsForTrack(track Track) []byte {
	var ret []byte

	for _, lyricsURL := range BuildFileNameForTrack(track) {
		songPage, err := goquery.NewDocument(AZLYRICS + lyricsURL + ".html")
		if err != nil {
			continue
		}
		song := strings.TrimSpace(songPage.Find("#main > b").First().Text())
		artist := songPage.Find("#main > h2").First().Text()
		artist = strings.Replace(artist, "LYRICS", "", -1)
		artist = strings.TrimSpace(artist)
		lyrics := strings.TrimSpace(songPage.Find("#main > div[style]").Text())
		ret = []byte(fmt.Sprintf("%s by %s\n\n%s", song, artist, lyrics))
	}

	return ret
}
