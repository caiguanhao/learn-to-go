package main

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
)

const (
	AZLYRICS = "http://www.azlyrics.com/lyrics/"
)

func (az AZLyrics) BuildFileName() ([]string, bool, bool) {
	az0 := func(input string) string {
		var re *regexp.Regexp

		input = strings.ToLower(input)
		input = strings.Replace(input, "p!nk", "pink", -1)

		re = regexp.MustCompile("(?i)f[uc*]{2}k") // fuck
		input = re.ReplaceAllString(input, "fuck")

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

	az3 := func(input string) string {
		var re *regexp.Regexp

		re = regexp.MustCompile("/.*")
		input = re.ReplaceAllString(input, "")

		re = regexp.MustCompile("&.*$")
		input = re.ReplaceAllString(input, "")

		input = az2(input)

		return input
	}

	base64enc := func(input string) string {
		return base64.StdEncoding.EncodeToString([]byte(input))
	}

	var artist, name string
	var validForAZLyrics bool = true
	var cacheable bool = true
	track := *az.track
	if len(track.Artist) == 0 {
		artist = "Unknown"
		cacheable = false
		validForAZLyrics = false
	} else {
		artist = az3(track.Artist)
	}
	if len(track.Name) == 0 {
		name = "Unknown"
		cacheable = false
		validForAZLyrics = false
	} else {
		name = track.Name
	}
	u1 := az1(name)
	u2 := az2(name)
	if len(artist) == 0 {
		artist = base64enc(track.Artist)
		validForAZLyrics = false
	}
	if len(u1) == 0 || len(u2) == 0 {
		u1 = base64enc(name)
		u2 = u1
		validForAZLyrics = false
	}
	ret := []string{
		fmt.Sprintf("%s/%s", artist, u1),
	}
	if u1 != u2 {
		ret = append(ret, fmt.Sprintf("%s/%s", artist, u2))
	}
	return ret, validForAZLyrics, cacheable
}

func (az AZLyrics) GetLyrics() []byte {
	var ret []byte

	lyricsURLs, validForAZLyrics, _ := az.BuildFileName()

	if !validForAZLyrics {
		return ret
	}

	for _, lyricsURL := range lyricsURLs {
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
		return ret
	}

	return ret
}
