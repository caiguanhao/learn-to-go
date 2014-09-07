package main

import (
	"bytes"
	"code.google.com/p/go.text/encoding/simplifiedchinese"
	"code.google.com/p/go.text/transform"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

/*
 * AZLYRICDBCN could not search single Chinese character, so use
 * AZLYRICDBPINYIN to search, and the two sites share the same
 * lyrics page ID
 */

const (
	AZLYRICDBPINYIN = "http://pinyin.azlyricdb.com"
	AZLYRICDBCN     = "http://cn.azlyricdb.com"
)

var (
	ArtistAliases = map[string]string{
		"Eason Chan": "陈奕迅",
	}
)

func (_ AZLyricDBCN) Convert(toUTF8 bool, input string) string {
	convert := func(input string, transformer transform.Transformer) string {
		r := transform.NewReader(bytes.NewReader([]byte(input)), transformer)
		ret, err := ioutil.ReadAll(r)
		if err != nil {
			return ""
		}
		return string(ret)
	}

	if toUTF8 {
		return convert(input, simplifiedchinese.GBK.NewDecoder())
	} else {
		return convert(input, simplifiedchinese.GBK.NewEncoder())
	}
}

func (az AZLyricDBCN) Search() *[]AZLyricDBCNResult {
	con := func(input string) string {
		return strings.TrimSpace(az.Convert(true, input))
	}

	results := &[]AZLyricDBCNResult{}

	var re *regexp.Regexp

	name := az.track.Name

	re = regexp.MustCompile("\\(.+?\\)") // (.*)
	name = strings.TrimSpace(re.ReplaceAllString(name, ""))
	name = Trad2SimpConvert(name)

	artist := az.track.Artist

	re = regexp.MustCompile("\\(.+?\\)") // (.*)
	artist = re.ReplaceAllString(artist, "")
	re = regexp.MustCompile("/.*")
	artist = strings.TrimSpace(re.ReplaceAllString(artist, ""))
	artist = Trad2SimpConvert(artist)

	alias := ArtistAliases[artist]
	if alias != "" {
		artist = alias
	}

	query := url.Values{"st": {"1"}, "search": {az.Convert(false, name)}}
	res, err := http.PostForm(fmt.Sprintf("%s/search", AZLYRICDBPINYIN), query)

	if err != nil {
		return results
	}

	doc, err := goquery.NewDocumentFromResponse(res)

	if err != nil {
		return results
	}

	doc.Find("a").Each(func(i int, anchor *goquery.Selection) {
		href, _ := anchor.Attr("href")
		if !strings.HasPrefix(href, "/lyrics/") {
			return
		}
		html, _ := anchor.Html()
		html = strings.Replace(con(html), "<br/>", " - ", -1)
		text := strings.Split(html, " - ")
		if len(text) < 2 {
			return
		}
		*results = append(*results, AZLyricDBCNResult{
			URL:    strings.TrimSpace(az.Convert(true, href)),
			Name:   strings.TrimSpace(text[0]),
			Artist: strings.TrimSpace(text[1]),
		})
	})

	explicitN := &[]AZLyricDBCNResult{}
	for _, result := range *results {
		if strings.EqualFold(result.Name, name) {
			*explicitN = append(*explicitN, result)
		}
	}
	if len(*explicitN) > 0 {
		*results = nil
		results = explicitN
	}

	explicitA := &[]AZLyricDBCNResult{}
	for _, result := range *results {
		if strings.EqualFold(result.Artist, artist) {
			*explicitA = append(*explicitA, result)
		}
	}
	if len(*explicitA) > 0 {
		*results = nil
		results = explicitA
	}

	return results
}

func (az AZLyricDBCN) GetLyrics() []byte {
	IsSpace := func(r rune) bool {
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ', '　', 0x85, 0xA0:
			return true
		}
		return false
	}

	con := func(input string) string {
		return strings.TrimFunc(az.Convert(true, input), IsSpace)
	}

	var ret []byte

	for _, result := range *az.Search() {
		lyricsURL := result.URL
		songPage, err := goquery.NewDocument(AZLYRICDBCN + lyricsURL)
		if err != nil {
			continue
		}

		info := strings.Split(con(songPage.Find("#dl > h1").First().Text()), "歌词 - ")
		song := info[0]
		artist := info[1]

		var lyrics string
		songPage.Find("#lrc > li").Each(func(i int, line *goquery.Selection) {
			lyrics += con(line.Text()) + "\n"
		})
		lyrics = strings.TrimSpace(lyrics)
		if len(lyrics) == 0 {
			continue
		}
		ret = []byte(fmt.Sprintf("%s by %s\n\n%s", song, artist, lyrics))
		return ret
	}

	return ret
}
