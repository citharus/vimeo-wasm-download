package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Url   string  `json:"url"`
	Size  int     `json:"size"`
}

type Video struct {
	Id              string    `json:"id"`
	BaseUrl         string    `json:"base_url"`
	Codecs          string    `json:"codecs"`
	Bitrate         int       `json:"bitrate"`
	AvgBitrate      int       `json:"avg_bitrate"`
	Duration        float64   `json:"duration"`
	Framerate       int       `json:"framerate"`
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
	Bitrate         int       `json:"bitrate"`
	AvgBitrate      int       `json:"avg_bitrate"`
	Duration        float64   `json:"duration"`
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

type MasterJson struct {
	ClipId  string  `json:"clip_id"`
	BaseUrl string  `json:"base_url"`
	Videos  []Video `json:"video"`
	Audios  []Audio `json:"audio"`
}

func GetMasterJson(url string) (*MasterJson, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	reader, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	mj := new(MasterJson)
	if err = json.Unmarshal(reader, mj); err != nil {
		return nil, err
	}
	fmt.Println("END of getmasterjson")
	return mj, nil
}
