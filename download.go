package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"syscall/js"
)

func DownloadVideo(videoId string, videoBuffer *bytes.Buffer, masterJson *MasterJson, baseUrl *url.URL) error {
	var video Video
	for _, v := range masterJson.Videos {
		if v.Id == videoId {
			video = v
			break
		}
	}

	is, err := base64.StdEncoding.DecodeString(video.InitSegment)
	if err != nil {
		return errors.New("failed to decode video init segment")
	}
	videoBuffer.Write(is)

	videoBaseUrl, _ := url.Parse(video.BaseUrl)
	for _, s := range video.Segments {
		segmentUrl, _ := url.Parse(s.Url)
		downloadUrl := baseUrl.ResolveReference(videoBaseUrl).ResolveReference(segmentUrl)
		res, err := http.Get(downloadUrl.String())
		if err != nil {
			return errors.New("failed to download video")
		}
		io.Copy(videoBuffer, res.Body)
		res.Body.Close()
	}

	videoDst := js.Global().Get("Uint8Array").New(videoBuffer.Len())
	js.CopyBytesToJS(videoDst, videoBuffer.Bytes())
	js.Global().Set("videoDst", videoDst)
	return nil
}

func DownloadAudio(audioId string, audioBuffer *bytes.Buffer, masterJson *MasterJson, baseUrl *url.URL) error {
	var audio Audio
	for _, a := range masterJson.Audios {
		if a.Id == audioId {
			audio = a
			break
		}
	}

	is, err := base64.StdEncoding.DecodeString(audio.InitSegment)
	if err != nil {
		return errors.New("failed to decode video init segment")
	}
	audioBuffer.Write(is)

	audioBaseUrl, _ := url.Parse(audio.BaseUrl)
	for _, s := range audio.Segments {
		segmentUrl, _ := url.Parse(s.Url)
		downloadUrl := baseUrl.ResolveReference(audioBaseUrl).ResolveReference(segmentUrl)
		res, err := http.Get(downloadUrl.String())
		if err != nil {
			return errors.New("failed to download video")
		}
		io.Copy(audioBuffer, res.Body)
		res.Body.Close()
	}

	audioDst := js.Global().Get("Uint8Array").New(audioBuffer.Len())
	js.CopyBytesToJS(audioDst, audioBuffer.Bytes())
	js.Global().Set("audioDst", audioDst)
	return nil
}
