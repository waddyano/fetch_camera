package main

import (
	"bytes"
	"fetch_camera/mjpeg"
	"flag"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "image/jpeg"
)

var interrupt bool = false

func setupCtrlCHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Printf("Ctrl/C\n")
		interrupt = true
	}()
}

func main() {
	setupCtrlCHandler()
	movie := flag.Bool("movie", false, "save as avi")
	spf := flag.Int("spf", 0, "set seconds per frame")
	baseurl := flag.String("url", "", "camera URL to fetch images from")
	fullurl := flag.String("fullurl", "", "camera URL to fetch images from")
	flag.Parse()
	if *baseurl == "" && *fullurl == "" {
		fmt.Fprintf(os.Stderr, "must specific base or full camera url")
		return
	}

	url := ""

	if *fullurl != "" {
		url = *fullurl
	} else {
		url = *baseurl + "/still"
	}

	shot := 1
	secperframe := 1
	if *spf != 0 {
		secperframe = *spf
	}
	ticker := time.NewTicker(time.Duration(secperframe) * time.Second)

	var aw mjpeg.AviWriter

	for !interrupt {
		select {
		case <-ticker.C:
			response, err := http.Get(url)
			if err != nil {
				panic(err)
			}

			fmt.Printf("status %d length %d type %s\n", response.StatusCode, response.ContentLength, response.Header.Get("Content-Type"))
			body := response.Body
			b, err := io.ReadAll(body)
			if err != nil {
				fmt.Printf("failed to read body\n")
				body.Close()
				continue
			}
			body.Close()
			if *movie {
				if aw == nil {
					config, format, err := image.DecodeConfig(bytes.NewReader(b))
					if err != nil {
						fmt.Printf("decode error %s\n", err.Error())
					} else {
						fmt.Printf("%s %d x %d\n", format, config.Width, config.Height)
					}
					aw, _ = mjpeg.New("test.avi", int32(config.Width), int32(config.Height), 5)
				} else {
					aw.AddFrame(b)
				}
			} else {
				f, err := os.Create(fmt.Sprintf("photo%d.jpg", shot))
				if err != nil {
					fmt.Printf("failed to create file\n")
					continue
				}
				shot++
				f.Write(b)
				f.Close()
			}
		}
	}

	if aw != nil {
		aw.Close()
	}
}
