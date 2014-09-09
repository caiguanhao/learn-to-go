package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
)

const (
	LYRICSMANIA = "http://www.lyricsmania.com/"
)

func (lm LyricsMania) BuildFileName() ([]string, bool) {
	az0 := func(input string) string {
		var re *regexp.Regexp

		input = strings.ToLower(input)
		input = strings.Replace(input, "p!nk", "pink", -1)
		input = strings.Replace(input, ".", "", -1)

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
		input = re.ReplaceAllString(input, "_")

		return input
	}

	az2 := func(input string) string {
		var re *regexp.Regexp

		input = az0(input)

		re = regexp.MustCompile("\\(.+?\\)") // (.*)
		input = re.ReplaceAllString(input, "")

		re = regexp.MustCompile("[^\\w]")
		input = re.ReplaceAllString(input, "_")

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

	sanitize := func(input string) string {
		re := regexp.MustCompile("_{1,}")
		input = re.ReplaceAllString(strings.TrimSpace(input), "_")
		input = strings.Trim(input, "_")
		return input
	}

	validForLyricsMania := true
	track := *lm.track
	artist := sanitize(az3(track.Artist))
	u1 := sanitize(az1(track.Name))
	u2 := sanitize(az2(track.Name))
	if len(artist) == 0 || len(u1) == 0 || len(u2) == 0 {
		validForLyricsMania = false
	}
	ret := []string{
		fmt.Sprintf("%s_lyrics_%s.html", u1, artist),
	}
	if u1 != u2 {
		ret = append(ret, fmt.Sprintf("%s_lyrics_%s.html", u2, artist))
	}
	return ret, validForLyricsMania
}

func (lm LyricsMania) GetLyrics() []byte {
	var ret []byte

	lyricsURLs, validForLyricsMania := lm.BuildFileName()

	if !validForLyricsMania {
		return ret
	}

	for _, lyricsURL := range lyricsURLs {
		songPage, err := goquery.NewDocument(LYRICSMANIA + lyricsURL)
		if err != nil {
			continue
		}
		song := strings.TrimSpace(songPage.Find(".lyrics-nav h2").First().Text())
		if len(song) < 7 {
			continue
		}
		song = song[0 : len(song)-7] // len(" lyrics") = 6
		artist := strings.TrimSpace(songPage.Find(".lyrics-nav h3").First().Text())
		body := songPage.Find(".lyrics-body").First()
		body.Get(0).RemoveChild(body.Find("#video-musictory").Get(0))
		body.Get(0).RemoveChild(body.Find("strong").First().Get(0))
		lyrics := strings.TrimSpace(body.Text())
		if len(song) == 0 || len(artist) == 0 || len(lyrics) == 0 {
			continue
		}
		ret = []byte(fmt.Sprintf("\"%s\" by %s\n\n%s", song, artist, lyrics))
		return ret
	}

	return ret
}
