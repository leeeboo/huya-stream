package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type GameStreamInfo struct {
	SCdnType      string `json:"sCdnType"`
	IIsMaster     int    `json:"iIsMaster"`
	LChannelId    int    `json:"lChannelId"`
	LSubChannelId int    `json:"lSubChannelId"`
	LPresenterUid int    `json:"lPresenterUid"`
	SStreamName   string `json:"sStreamName"`
	SHlsUrl       string `json:"sHlsUrl"`
	SHlsUrlSuffix string `json:"sHlsUrlSuffix"`
	SHlsAntiCode  string `json:"sHlsAntiCode"`
}

type GameLiveInfo struct {
	Nick string `json:"nick"`
}

type StreamInfo struct {
	GameLiveInfo       *GameLiveInfo    `json:"gameLiveInfo"`
	GameStreamInfoList []GameStreamInfo `json:"gameStreamInfoList"`
}

type MultiStreamInfo struct {
	SDisplayName string `json:"sDisplayName"`
	IBitRate     int    `json:"iBitRate"`
}

type Stream struct {
	Status           int               `json:"status"`
	Msg              string            `json:"msg"`
	Data             []StreamInfo      `json:"data"`
	VMultiStreamInfo []MultiStreamInfo `json:"vMultiStreamInfo"`
}

type HyPlayerConfig struct {
	Html5     int     `json:"html5"`
	WEBYYHOST string  `json:"WEBYYHOST"`
	WEBYYSWF  string  `json:"WEBYYSWF"`
	WEBYYFROM string  `json:"WEBYYFROM"`
	Vappid    int     `json:"vappid"`
	Stream    *Stream `json:"stream"`
}

func handler(w http.ResponseWriter, r *http.Request) {

	room := r.URL.Query().Get("room")
	room = strings.TrimSpace(room)

	m3u8, err := getM3u8(room)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, m3u8, 302)
	//fmt.Fprintf(w, m3u8)
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getM3u8(room string) (string, error) {

	if room == "" {
		return "", errors.New("Room name invalid.")
	}

	api := fmt.Sprintf("https://www.huya.com/%s", room)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	resp, err := http.Get(api)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	htmlStr := string(body)

	tmp := strings.Split(htmlStr, "hyPlayerConfig =")

	if len(tmp) < 2 {
		return "", errors.New("Parse error.")
	}

	tmp = strings.Split(tmp[1], "window.TT_LIVE_TIMING")

	if len(tmp) < 2 {
		return "", errors.New("Parse error.")
	}

	jsonStr := strings.Replace(tmp[0], "};", "}", 1)

	var hyPlayerConfig HyPlayerConfig

	err = json.Unmarshal([]byte(jsonStr), &hyPlayerConfig)

	if err != nil {
		return "", err
	}

	if hyPlayerConfig.Stream == nil {
		return "", errors.New("No live stream.")
	}

	if len(hyPlayerConfig.Stream.Data) <= 0 {
		return "", errors.New("No live stream.")
	}

	var m3u8 string

	for _, v := range hyPlayerConfig.Stream.Data[0].GameStreamInfoList {
		if v.SHlsUrlSuffix == "m3u8" {
			m3u8 = fmt.Sprintf("%s/%s.%s", v.SHlsUrl, v.SStreamName, v.SHlsUrlSuffix)
		}
	}

	if m3u8 == "" {
		return "", errors.New("Parse error.")
	}

	return m3u8, nil
}
