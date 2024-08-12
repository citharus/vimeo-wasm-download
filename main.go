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
	Url        *url.URL
	MasterJson *MasterJson
	VideoId    string
	AudioId    string
}

func goToNextStep(currentButton js.Value) {
	currentButton.
		Get("parentElement").
		Get("nextElementSibling").
		Get("classList").
		Call("remove", "invisible")
	currentButton.Set("disabled", "disabled")
	currentButton.Get("parentElement").Set("disabled", "disabled")
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

func main() {
	firstStepEvent := js.FuncOf(func(this js.Value, args []js.Value) any {
		go func() error {
			i := args[0].
				Get("target").
				Get("parentElement").
				Call("querySelector", "input[type='url']")

			if i.Get("value").String() == "" {
				displayTempError("No URL was given")
				return nil
			}

			u, err := url.Parse(i.Get("value").String())
			if err != nil {
				displayTempError("Not a real URL")
				return err
			}
			data.Url = u

			masterJson, err := GetMasterJson(u.String())
			if err != nil {
				displayTempError("Failed to download master.json")
				return err
			}
			data.MasterJson = masterJson

			sort.Slice(masterJson.Videos, func(i, j int) bool {
				return masterJson.Videos[i].Height < masterJson.Videos[j].Height
			})

			sort.Slice(masterJson.Audios, func(i, j int) bool {
				sizeI, _ := masterJson.Audios[i].GetSize()
				sizeJ, _ := masterJson.Audios[j].GetSize()
				return sizeI < sizeJ
			})

			n := args[0].Get("target").Get("parentElement").Get("nextElementSibling")

			form := js.Global().Get("document").Call("createElement", "form")
			bb := n.Call("getElementsByTagName", "button").Index(0)
			n.Call("insertBefore", form, bb)
			for i, video := range masterJson.Videos {
				size, err := video.GetSize()
				if err != nil {
					displayTempError(err.Error())
					return err
				}

				label := js.Global().Get("document").Call("createElement", "label")
				label.Set("htmlFor", video.Id)
				label.Set("innerText", fmt.Sprintf("Video %d: %dp (%s)", i, video.Height, toHumanReadableSize(float64(size))))
				form.Call("appendChild", label)

				input := js.Global().Get("document").Call("createElement", "input")
				input.Set("value", video.Id)
				input.Set("id", video.Id)
				input.Set("type", "radio")
				input.Set("name", "radio")
				label.Call("prepend", input)
			}

			form = js.Global().Get("document").Call("createElement", "form")
			bb = n.Get("nextElementSibling").Call("getElementsByTagName", "button").Index(0)
			n.Get("nextElementSibling").Call("insertBefore", form, bb)
			for i, audio := range masterJson.Audios {
				size, err := audio.GetSize()
				if err != nil {
					displayTempError(err.Error())
					return err
				}

				label := js.Global().Get("document").Call("createElement", "label")
				label.Set("htmlFor", audio.Id)
				label.Set("innerText", fmt.Sprintf("Audio %d: %s", i, toHumanReadableSize(float64(size))))
				form.Call("appendChild", label)

				input := js.Global().Get("document").Call("createElement", "input")
				input.Set("value", audio.Id)
				input.Set("id", audio.Id)
				input.Set("type", "radio")
				input.Set("name", "radio")
				label.Call("prepend", input)
			}

			goToNextStep(args[0].Get("target"))
			i.Set("disabled", "disabled")
			return nil
		}()
		return nil
	})

	secondStepEvent := js.FuncOf(func(this js.Value, args []js.Value) any {
		selectedVideo := args[0].
			Get("target").
			Get("parentElement").
			Call("querySelector", "input[type='radio']:checked")

		if selectedVideo.IsNull() {
			displayTempError("No video was selected")
			return nil
		}

		data.VideoId = selectedVideo.Get("value").String()

		goToNextStep(args[0].Get("target"))
		return nil
	})

	thirdStepEvent := js.FuncOf(func(this js.Value, args []js.Value) any {
		selectedAudio := args[0].
			Get("target").
			Get("parentElement").
			Call("querySelector", "input[type='radio']:checked")

		if selectedAudio.IsNull() {
			displayTempError("No audio was selected")
			return nil
		}

		data.AudioId = selectedAudio.Get("value").String()

		go func() {
			baseUrl, _ := url.Parse(data.MasterJson.BaseUrl)
			data.Url = data.Url.ResolveReference(baseUrl)

			var wg sync.WaitGroup
			wg.Add(2)

			var audioBuffer, videoBuffer bytes.Buffer
			go func() {
				DownloadVideo(data.VideoId, &videoBuffer, data.MasterJson, data.Url)
				wg.Done()
			}()
			go func() {
				DownloadAudio(data.AudioId, &audioBuffer, data.MasterJson, data.Url)
				wg.Done()
			}()

			wg.Wait()
			js.Global().Call("combine")
		}()

		goToNextStep(args[0].Get("target"))
		return nil
	})

	buttons.Index(0).Call("addEventListener", "click", firstStepEvent)
	buttons.Index(1).Call("addEventListener", "click", secondStepEvent)
	buttons.Index(2).Call("addEventListener", "click", thirdStepEvent)

	select {}
}
