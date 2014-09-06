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

func (az AZLyrics) BuildFileName() []string {
	az0 := func(input string) string {
		var re *regexp.Regexp

		input = strings.ToLower(input)
		input = strings.Replace(input, "p!nk", "pink", -1)

		re = regexp.MustCompile("(?i)f[uc*]{2}k") // fuck
		input = re.ReplaceAllString(input, "")

		re = regexp.MustCompile("\\[.+?\\]") // [.*]
		input = re.ReplaceAllString(input, "")

		return input
	}

	az1 := func(input string) string {
		var re *regexp.Regexp

		input = az0(input)

		re = regexp.MustCompile("[^\\w]")
		input = re.ReplaceAllString(input, "")

		return input
	}

	az2 := func(input string) string {
		var re *regexp.Regexp

		input = az0(input)

		re = regexp.MustCompile("\\(.+?\\)") // (.*)
		input = re.ReplaceAllString(input, "")

		re = regexp.MustCompile("[^\\w]")
		input = re.ReplaceAllString(input, "")

		return input
	}

	track := *az.track
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

func (az AZLyrics) GetLyrics() []byte {
	var ret []byte

	for _, lyricsURL := range az.BuildFileName() {
		songPage, err := goquery.NewDocument(AZLYRICS + lyricsURL + ".html")
		if err != nil {
			continue
		}
		song := strings.TrimSpace(songPage.Find("#main > b").First().Text())
		artist := songPage.Find("#main > h2").First().Text()
		artist = strings.Replace(artist, "LYRICS", "", -1)
		artist = strings.TrimSpace(artist)
		lyrics := strings.TrimSpace(songPage.Find("#main > div[style]").Text())
		if len(song) == 0 || len(artist) == 0 || len(lyrics) == 0 {
			continue
		}
		ret = []byte(fmt.Sprintf("%s by %s\n\n%s", song, artist, lyrics))
	}

	return ret
}
