package structs

import (
	"strings"
)

type (
	Result struct {
		URL       string
		TrackName string
	}

	Results []Result
)

func (results Results) FilterByTrack(track *Track) string {
	if len(results) < 1 {
		return ""
	}
	var index int = 0
	for i, result := range results {
		if lowerCaseNoSpace(result.TrackName) == lowerCaseNoSpace((*track).Name) {
			index = i
			break
		}
	}
	return results[index].URL
}

// you can test with `lyrics --no-pager Mean Pink`
func (results Results) FilterByQuery(query string) string {
	if len(results) < 1 {
		return ""
	}
	var index int = 0
	var max float64
	parts := strings.Split(query, " ")

	for i, result := range results {
		for _, part := range parts {
			per := score(lowerCaseNoSpace(result.TrackName), lowerCaseNoSpace(part))
			if per > max {
				max = per
				index = i
			}
		}
	}

	return results[index].URL
}

func score(long, short string) float64 {
	return 1 - float64(len(strings.Replace(long, short, "", -1)))/float64(len(long))
}
