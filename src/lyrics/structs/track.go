package structs

import (
	"regexp"
	"strings"
)

type Track struct {
	Name   string
	Artist string
}

func (track Track) Convert() *Track {
	name := track.Name
	re := regexp.MustCompile("(\\[|\\().+?(\\]|\\))") // (.*) [.*]
	name = re.ReplaceAllString(name, "")
	re = regexp.MustCompile("(?i)f[uc*]{2}k") // fuck
	name = re.ReplaceAllString(name, "")

	artist := track.Artist
	artist = strings.Replace(artist, "!", "i", -1) // P!nk

	return &Track{
		Name:   name,
		Artist: artist,
	}
}

func (track Track) FileName() *Track {
	converted := track.Convert()
	return &Track{
		Name:   lowerCaseNoSpace(converted.Name),
		Artist: lowerCaseNoSpace(converted.Artist),
	}
}

func (track Track) Query() string {
	if &track == nil {
		return ""
	}
	converted := track.Convert()
	return (*converted).Name + " " + (*converted).Artist
}

func (trackA Track) Equal(trackB Track) bool {
	return trackA.Name == trackB.Name && trackA.Artist == trackB.Artist
}

func (trackA Track) NotEqual(trackB Track) bool {
	return !trackA.Equal(trackB)
}

func lowerCaseNoSpace(input string) string {
	return strings.Replace(strings.ToLower(input), " ", "", -1)
}
