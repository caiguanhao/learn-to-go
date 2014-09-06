package main

type (
	Provider interface {
		GetLyrics() []byte
	}

	AZLyrics struct {
		track *Track
	}

	AZLyricDBCNResult struct {
		URL    string
		Name   string
		Artist string
	}

	AZLyricDBCN struct {
		track *Track
	}

	Track struct {
		Name   string
		Artist string
		AZLyrics
		AZLyricDBCN
	}
)

func NewTrack(Name, Artist string) *Track {
	newTrack := &Track{
		Name:        Name,
		Artist:      Artist,
		AZLyrics:    AZLyrics{},
		AZLyricDBCN: AZLyricDBCN{},
	}
	(*newTrack).AZLyrics.track = newTrack
	(*newTrack).AZLyricDBCN.track = newTrack
	return newTrack
}

func (trackA Track) Equal(trackB Track) bool {
	return trackA.Name == trackB.Name && trackA.Artist == trackB.Artist
}

func (trackA Track) NotEqual(trackB Track) bool {
	return !trackA.Equal(trackB)
}
