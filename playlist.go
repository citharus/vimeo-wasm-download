package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
)

type Segment struct {
	Url  string `json:"url"`
	Size int    `json:"size"`
}

type Video struct {
	Id              string    `json:"id"`
	BaseUrl         string    `json:"base_url"`
	Codecs          string    `json:"codecs"`
	Height          int       `json:"height"`
	IndexSegmentURL string    `json:"index_segment"`
	InitSegment     string    `json:"init_segment"`
	Segments        []Segment `json:"segments"`
}

func (video *Video) GetSize() (int, error) {
	var size int
	for _, segment := range video.Segments {
		size += segment.Size
	}
	is, err := base64.StdEncoding.DecodeString(video.InitSegment)
	if err != nil {
		return 0, err
	}
	size += len(is)
	return size, nil
}

type Audio struct {
	Id              string    `json:"id"`
	BaseUrl         string    `json:"base_url"`
	Codecs          string    `json:"codecs"`
	IndexSegmentURL string    `json:"index_segment"`
	InitSegment     string    `json:"init_segment"`
	Segments        []Segment `json:"segments"`
}

func (audio *Audio) GetSize() (int, error) {
	var size int
	for _, segment := range audio.Segments {
		size += segment.Size
	}
	is, err := base64.StdEncoding.DecodeString(audio.InitSegment)
	if err != nil {
		return 0, err
	}
	size += len(is)
	return size, nil
}

type Playlist struct {
	ClipId  string  `json:"clip_id"`
	BaseUrl string  `json:"base_url"`
	Videos  []Video `json:"video"`
	Audios  []Audio `json:"audio"`
}

func GetPlaylist(url string) (*Playlist, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	reader, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	mj := new(Playlist)
	if err = json.Unmarshal(reader, mj); err != nil {
		return nil, err
	}
	return mj, nil
}
