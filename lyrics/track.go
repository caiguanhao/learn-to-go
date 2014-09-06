package main

type (
	AZLyrics struct {
		track *Track
	}

	Track struct {
		Name   string
		Artist string
		AZLyrics
	}
)

func NewTrack(Name, Artist string) *Track {
	newTrack := &Track{
		Name:     Name,
		Artist:   Artist,
		AZLyrics: AZLyrics{},
	}
	(*newTrack).AZLyrics.track = newTrack
	return newTrack
}

func (trackA Track) Equal(trackB Track) bool {
	return trackA.Name == trackB.Name && trackA.Artist == trackB.Artist
}

func (trackA Track) NotEqual(trackB Track) bool {
	return !trackA.Equal(trackB)
}
