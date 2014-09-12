package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

type (
	Provider interface {
		GetLyrics() []byte
	}

	TrackPointer struct {
		track *Track
	}

	AZLyrics     TrackPointer
	AZLyricDBCN  TrackPointer
	LyricsMania  TrackPointer
	SongMeanings TrackPointer
	ITunes       TrackPointer

	AZLyricDBCNResult struct {
		URL    string
		Name   string
		Artist string
	}

	Track struct {
		Name   string
		Artist string
		ITunes
		AZLyrics
		AZLyricDBCN
		LyricsMania
		SongMeanings
	}
)

const (
	OSASCRIPT = `
	if application "iTunes" is running then
		tell application "iTunes"
			(get name of current track) & "\n" & (get artist of current track)
		end tell
	end if`
)

func NewTrack(Name, Artist string) *Track {
	newTrack := &Track{
		Name:         Name,
		Artist:       Artist,
		ITunes:       ITunes{},
		AZLyrics:     AZLyrics{},
		AZLyricDBCN:  AZLyricDBCN{},
		LyricsMania:  LyricsMania{},
		SongMeanings: SongMeanings{},
	}
	(*newTrack).ITunes.track = newTrack
	(*newTrack).AZLyrics.track = newTrack
	(*newTrack).AZLyricDBCN.track = newTrack
	(*newTrack).LyricsMania.track = newTrack
	(*newTrack).SongMeanings.track = newTrack
	return newTrack
}

func (iTunes ITunes) GetCurrentTrack() (bool, error) {
	var output bytes.Buffer
	cmd := exec.Command("osascript")
	cmd.Stdin = strings.NewReader(OSASCRIPT)
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return false, err
	}
	var out string
	out = strings.TrimSpace(output.String())
	if out == "" {
		return false, errors.New("empty content")
	}
	info := strings.Split(out, "\n")
	if len(info) != 2 {
		return false, errors.New("invalid data")
	}
	name := strings.TrimSpace(info[0])
	artist := strings.TrimSpace(info[1])
	track := iTunes.track
	if track.Name != name || track.Artist != artist {
		track.Name = name
		track.Artist = artist
		return true, nil
	}
	return false, nil
}
