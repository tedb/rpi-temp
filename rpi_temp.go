package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

func main() {
	urlTemplate := os.Getenv("ADAFRUIT_IO_URL")
	apiKey := os.Getenv("ADAFRUIT_IO_KEY")

	probeDirs, err := filepath.Glob("/sys/bus/w1/devices/28-*")
	if err != nil {
		panic(err)
	}

	g, ctx := errgroup.WithContext(context.Background())

	for _, dirName := range probeDirs {
		dirName := dirName

		g.Go(func() error {
			return processDir(ctx, dirName, urlTemplate, apiKey)
		})
	}

	if err := g.Wait(); err != nil {
		if err != nil {
			panic(err)
		}
	}
}

func processDir(ctx context.Context, dirName string, urlTemplate, apiKey string) (err error) {
	name, degC, degF, err := readTemp(dirName)
	if err != nil {
		return
	}

	fmt.Printf("%s: %.2fc %.2ff\n", name, degC, degF)

	url := fmt.Sprintf(urlTemplate, url.PathEscape(name))

	err = postAdafruitValue(ctx, url, apiKey, degC)
	if err != nil {
		return
	}

	return
}

func readTemp(dirName string) (name string, degC, degF float64, err error) {
	rawName, err := ioutil.ReadFile(filepath.Join(dirName, "name"))
	if err != nil {
		return
	}

	name = strings.TrimPrefix(strings.TrimSpace(string(rawName)), "28-")

	rawMilliDegC, err := ioutil.ReadFile(filepath.Join(dirName, "temperature"))
	if err != nil {
		return
	}

	milliDegC, err := strconv.ParseFloat(strings.TrimSpace(string(rawMilliDegC)), 32)
	if err != nil {
		return
	}

	degC = milliDegC / 1000.0 //nolint:gomnd
	degF = degC*1.8 + 32      //nolint:gomnd

	return
}

func postAdafruitValue(ctx context.Context, urlStr, key string, value float64) (err error) {
	data := url.Values{}
	data.Set("value", fmt.Sprintf("%f", value))
	data.Set("created_at", time.Now().Format(time.RFC3339))

	r, _ := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, strings.NewReader(data.Encode()))
	r.Header.Add("X-AIO-Key", key)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	client := &http.Client{}

	resp, err := client.Do(r)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	fmt.Println(resp.Status, string(body))

	return
}
