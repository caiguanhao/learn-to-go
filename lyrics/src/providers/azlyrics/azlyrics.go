package azlyrics

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/url"
	"strings"
	"structs"
)

func Search(__query__ string) []structs.Result {
	results := []structs.Result{}
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
		return results
	}
	doc.Find("a").Each(func(i int, anchor *goquery.Selection) {
		href, _ := anchor.Attr("href")
		if strings.HasPrefix(href, "http://www.azlyrics.com/lyrics/") {
			results = append(results, structs.Result{
				URL:       href,
				TrackName: anchor.Text(),
			})
		}
	})
	return results
}

func SearchByTrack(track *structs.Track) []structs.Result {
	return Search((*track).Query())
}

func GetLyrics(lyricsURL string) []byte {
	songPage, err := goquery.NewDocument(lyricsURL)
	if err != nil {
		return []byte{}
	}
	song := strings.TrimSpace(songPage.Find("#main > b").First().Text())
	artist := songPage.Find("#main > h2").First().Text()
	artist = strings.Replace(artist, "LYRICS", "", -1)
	artist = strings.TrimSpace(artist)
	lyrics := strings.TrimSpace(songPage.Find("#main > div[style]").Text())
	return []byte(fmt.Sprintf("%s by %s\n\n%s", song, artist, lyrics))
}
