//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"syscall/js"
	"time"
)

type Data struct {
	Url      *url.URL
	Playlist *Playlist
	VideoId  string
	AudioId  string
}

func toHumanReadableSize(size float64) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	i := 0
	for size >= 1024 && i < len(sizes) {
		size = size / 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", size, sizes[i])
}

func displayTempError(msg string) {
	go func() {
		div := js.Global().Get("document").Call("createElement", "div")
		div.Set("innerText", msg)
		defer div.Call("remove")
		js.Global().Get("document").Call("getElementById", "errors").Call("prepend", div)
		time.Sleep(2 * time.Second)
	}()
}

var buttons = js.Global().Get("document").Call("getElementsByTagName", "button")
var data = new(Data)

func createInputAndAppend(form js.Value, label, id string) {
	i := js.Global().Get("document").Call("createElement", "input")
	i.Set("value", id)
	i.Set("id", id)
	i.Set("type", "radio")
	i.Set("name", "radio")

	l := js.Global().Get("document").Call("createElement", "label")
	l.Set("htmlFor", id)
	l.Set("innerText", label)

	form.Call("appendChild", i)
	form.Call("appendChild", l)
}

func main() {
	processPlaylistUrl := js.FuncOf(func(this js.Value, args []js.Value) any {
		go func() error {
			playlistUrlInput := args[0].
				Get("target").
				Get("parentElement").
				Call("querySelector", "input[type='url']")
			if playlistUrlInput.Get("value").String() == "" {
				displayTempError("No URL")
				return nil
			}

			u, err := url.Parse(playlistUrlInput.Get("value").String())
			if err != nil {
				displayTempError("Not a real URL")
				return err
			}
			data.Url = u

			playlist, err := GetPlaylist(u.String())
			if err != nil {
				displayTempError("Failed to download playlist.json")
				return err
			}
			data.Playlist = playlist

			baseUrl, _ := url.Parse(data.Playlist.BaseUrl)
			data.Url = data.Url.ResolveReference(baseUrl)

			args[0].Get("target").Set("disabled", "disabled")
			playlistUrlInput.Set("disabled", "disabled")

			sort.Slice(playlist.Videos, func(i, j int) bool {
				return playlist.Videos[i].Height < playlist.Videos[j].Height
			})

			sort.Slice(playlist.Audios, func(i, j int) bool {
				sizeI, _ := playlist.Audios[i].GetSize()
				sizeJ, _ := playlist.Audios[j].GetSize()
				return sizeI < sizeJ
			})

			videoForm := js.Global().Get("document").Call("getElementById", "video-form")

			for _, video := range playlist.Videos {
				//size, err := video.GetSize()
				if err != nil {
					displayTempError(err.Error())
					return err
				}

				createInputAndAppend(videoForm, fmt.Sprintf("%dp", video.Height), video.Id)
			}

			audioForm := js.Global().Get("document").Call("getElementById", "audio-form")

			for _, audio := range playlist.Audios {
				size, err := audio.GetSize()
				if err != nil {
					displayTempError(err.Error())
					return err
				}

				createInputAndAppend(audioForm, fmt.Sprintf("%s", toHumanReadableSize(float64(size))), audio.Id)
			}

			js.Global().Get("document").
				Call("getElementById", "video").
				Get("classList").
				Call("remove", "hidden")
			return nil
		}()
		return nil
	})

	downloadVideo := js.FuncOf(func(this js.Value, args []js.Value) any {
		go func() error {
			selectedResolution := args[0].
				Get("target").
				Get("parentElement").
				Call("querySelector", "input[type='radio']:checked")

			if selectedResolution.IsNull() {
				displayTempError("No video was selected")
				return nil
			}

			data.VideoId = selectedResolution.Get("value").String()

			js.Global().Get("document").
				Call("getElementById", "audio").
				Get("classList").
				Call("remove", "hidden")

			js.Global().
				Get("document").
				Call("getElementById", "video-progress").
				Get("parentElement").
				Get("style").
				Set("display", "block")

			js.Global().
				Get("document").
				Call("getElementById", "video-form").
				Get("style").
				Set("display", "none")

			args[0].Get("target").Get("style").Set("display", "none")

			js.Global().
				Get("document").
				Call("getElementsByTagName", "h4").
				Index(1).
				Set("innerHTML", "downloading video")

			go func() {
				var wg sync.WaitGroup
				var videoBuffer bytes.Buffer
				var pChan = make(chan int)
				go func() {
					wg.Add(1)
					defer close(pChan)
					err := DownloadVideo(data.VideoId, &videoBuffer, data.Playlist, data.Url, pChan)
					if err != nil {
						displayTempError(err.Error())
						return
					}
					wg.Done()
				}()

				vp := js.Global().Get("document").Call("getElementById", "video-progress")
				for p := range pChan {
					vp.Get("style").Set("width", fmt.Sprintf("%d%%", p))
				}
				wg.Wait()

				vp.Get("style").Set("background-color", "#4ade80")
				js.Global().
					Get("document").
					Call("getElementsByTagName", "h4").
					Index(1).
					Set("innerHTML", "finished downloading video")
			}()
			return nil
		}()
		return nil
	})

	downloadAudio := js.FuncOf(func(this js.Value, args []js.Value) any {
		go func() error {
			selectedQuality := args[0].
				Get("target").
				Get("parentElement").
				Call("querySelector", "input[type='radio']:checked")

			if selectedQuality.IsNull() {
				displayTempError("No audio was selected")
				return nil
			}

			data.AudioId = selectedQuality.Get("value").String()

			js.Global().
				Get("document").
				Call("getElementById", "audio-progress").
				Get("parentElement").
				Get("style").
				Set("display", "block")
			js.Global().
				Get("document").
				Call("getElementById", "audio-form").
				Get("style").
				Set("display", "none")
			args[0].Get("target").Get("style").Set("display", "none")

			js.Global().
				Get("document").
				Call("getElementsByTagName", "h4").
				Index(2).
				Set("innerHTML", "downloading audio")

			go func() {
				var wg sync.WaitGroup
				var audioBuffer bytes.Buffer
				var pCHan = make(chan int)
				go func() {
					wg.Add(1)
					defer close(pCHan)
					err := DownloadAudio(data.AudioId, &audioBuffer, data.Playlist, data.Url, pCHan)
					if err != nil {
						displayTempError(err.Error())
						return
					}
					wg.Done()
				}()

				vp := js.Global().Get("document").Call("getElementById", "audio-progress")
				for p := range pCHan {
					vp.Get("style").Set("width", fmt.Sprintf("%d%%", p))
				}
				wg.Wait()

				vp.Get("style").Set("background-color", "#4ade80")
				js.Global().
					Get("document").
					Call("getElementsByTagName", "h4").
					Index(2).
					Set("innerHTML", "finished downloading audio")

				js.Global().Call("combine")

				js.Global().Get("document").
					Call("getElementById", "download").
					Get("classList").
					Call("remove", "hidden")

				js.Global().Call("combine")
			}()
			return nil
		}()
		return nil
	})

	buttons.Index(0).Call("addEventListener", "click", processPlaylistUrl)
	buttons.Index(1).Call("addEventListener", "click", downloadVideo)
	buttons.Index(2).Call("addEventListener", "click", downloadAudio)

	select {}
}
