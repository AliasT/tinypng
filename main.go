package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// TinyPNG ...
type TinyPNG struct {
	authorization string
}

// TinyPNGOutput ...
type TinyPNGOutput struct {
	URL string `json:"url"`
}

// TinyPNGResponse ...
type TinyPNGResponse struct {
	Output TinyPNGOutput `json:"output"`
}

// TargetURL ...
const TargetURL = "https://api.tinify.com/shrink"

// authorization
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// PostFile ...
func (tiny *TinyPNG) PostFile(filename string, wg *sync.WaitGroup) error {
	defer wg.Done()

	bodyBuf := &bytes.Buffer{}
	fh, err := os.Open(filename)

	if err != nil {
		fmt.Println("error opening file")
		return err
	}

	defer fh.Close()

	_, err = io.Copy(bodyBuf, fh)

	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", TargetURL, bodyBuf)

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("authorization", tiny.authorization)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	response, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var tinyResponse TinyPNGResponse

	if err = json.Unmarshal(response, &tinyResponse); err != nil {
		return err
	}

	tiny.DownloadFile(filename, tinyResponse.Output.URL)

	return nil
}

// DownloadFile ...
func (tiny *TinyPNG) DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {
	var target string

	if len(os.Args) < 2 {
		target = "."
	} else {
		target = os.Args[1]
	}

	if os.Getenv("TINY_PNG_KEY") == "" {
		log.Fatalln("please provide a tiny png api key")
	}

	tiny := TinyPNG{
		"Basic " + basicAuth("api", os.Getenv("TINY_PNG_KEY")),
	}

	var wg sync.WaitGroup

	err := filepath.Walk(target,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// 绝对链接
			fullpath, _ := filepath.Abs(path)
			wg.Add(1)
			go tiny.PostFile(fullpath, &wg)
			return nil
		})

	wg.Wait()
	if err != nil {
		log.Println(err)
	}
}
