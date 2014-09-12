package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/url"
	"regexp"
	"strings"
)

func (sm SongMeanings) Search() []string {
	sm0 := func(input string) string {
		var re *regexp.Regexp

		re = regexp.MustCompile("\\(.+?\\)") // (.*)
		input = re.ReplaceAllString(input, "")

		re = regexp.MustCompile("\\[.+?\\]") // [.*]
		input = re.ReplaceAllString(input, "")

		re = regexp.MustCompile("/.*")
		input = re.ReplaceAllString(input, "")

		re = regexp.MustCompile("[^\\w]+")
		input = re.ReplaceAllString(input, " ")

		return input
	}

	results := []string{}
	query := url.Values{}
	query.Add("query", fmt.Sprintf("%s %s", sm0((*sm.track).Artist),
		sm0((*sm.track).Name)))
	query.Add("type", "songtitles")
	URL := url.URL{
		Scheme:   "http",
		Host:     "songmeanings.com",
		Path:     "/query/",
		RawQuery: query.Encode(),
	}
	doc, err := goquery.NewDocument(URL.String())
	if err != nil {
		return results
	}
	if doc.Find("#content .lyric-box").Length() == 1 { // redirected to lyrics page
		return []string{URL.String()}
	}
	doc.Find("tr.item").Each(func(i int, row *goquery.Selection) {
		songAnchor := row.Find("a").Eq(0)
		songAnchorHref, _ := songAnchor.Attr("href")
		songName := strings.TrimSpace(songAnchor.Text())
		if strings.EqualFold(songName, (*sm.track).Name) ||
			strings.EqualFold(songName, sm0((*sm.track).Name)) {
			artistAnchor := row.Find("a").Eq(1)
			artistName := strings.ToLower(strings.TrimSpace(artistAnchor.Text()))
			_artistName := strings.ToLower((*sm.track).Artist)
			if strings.Contains(_artistName, artistName) ||
				strings.Contains(artistName, _artistName) {
				results = append(results, songAnchorHref)
			}
		}
	})
	return results
}

func (sm SongMeanings) GetLyrics() []byte {
	var ret []byte

	for _, lyricsURL := range sm.Search() {
		songPage, err := goquery.NewDocument(lyricsURL)
		if err != nil {
			continue
		}
		breadcrumbs := songPage.Find(".breadcrumbs > li")
		artist := strings.TrimSpace(breadcrumbs.Eq(1).Text())
		song := strings.TrimSpace(breadcrumbs.Last().Text())
		body := songPage.Find("#content .lyric-box")
		body.Get(0).RemoveChild(body.Find("div").Last().Get(0))
		lyrics := strings.TrimSpace(body.Text())
		if len(song) == 0 || len(artist) == 0 || len(lyrics) == 0 {
			continue
		}
		lyricsLines := strings.Split(lyrics, "\n")
		var trimed []string
		for _, line := range lyricsLines {
			trimed = append(trimed, strings.TrimSpace(line))
		}
		lyrics = strings.Join(trimed, "\n")
		ret = []byte(fmt.Sprintf("\"%s\" by %s\n\n%s", song, artist, lyrics))
		return ret
	}

	return ret
}
