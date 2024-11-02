package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
)

const (
	archNewsURL = "https://archlinux.org/feeds/news/"
	// archNewsURL   = "http://localhost:8080/fakenews.xml"
	storeFile     = "/var/lib/breakcheck.json"
	writeFileMode = 0o600
)

const (
	RetrieveNewsDataFailedErrorCode      = 100
	DecodeNewsDataFailedErrorCode        = 101
	LoadSeenItemsFromFileFailedErrorCode = 102
)

type Rss struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	LastBuildDate string `xml:"lastBuildDate"`
	Description   string `xml:"description"`
	Items         []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

func main() {
	slog.Debug("about to retrieve news data")
	newsData, err := retrieveNewsData()
	if err != nil {
		fmt.Printf("could not get news data: %v\n", err)
		os.Exit(RetrieveNewsDataFailedErrorCode)
	}

	slog.Debug("about to decode RSS")
	rss, err := decodeRss(newsData)
	if err != nil {
		fmt.Printf("could not decode news data: %v\n", err)
		os.Exit(DecodeNewsDataFailedErrorCode)
	}

	slog.Debug("about to load seen items")
	seenItems, err := loadSeenItemsFromFile(storeFile)
	if err != nil {
		fmt.Printf("could not decode news data: %v\n", err)
		os.Exit(LoadSeenItemsFromFileFailedErrorCode)
	}

	unseenItems := make([]string, 0)
	items := make([]string, 0, len(rss.Channel.Items))

	slog.Debug("process rss items")
	for _, item := range rss.Channel.Items {
		if !slices.Contains(seenItems, item.Link) {
			slog.Debug("found unseen item, storing", "title", item.Title, "link", item.Link)
			unseenItems = append(unseenItems, item.Link)
		}
		items = append(items, item.Link)
	}

	if len(unseenItems) == 0 {
		os.Exit(0)

		return
	}

	err = saveSeenItemsToFile(items, storeFile)
	if err != nil {
		fmt.Printf("could not save seen newHashedItems to file: %v\n", err)
	}

	fmt.Printf("There have been one or more unread news items:\n")
	for _, item := range unseenItems {
		fmt.Printf("Link: %s\n", item)
	}
	fmt.Printf("\n")
	fmt.Printf("Make sure you have read the news items and restart the upgrade to complete.\n")
	fmt.Printf("\n")

	os.Exit(1)
}

func retrieveNewsData() ([]byte, error) {
	slog.Debug("retrieving news")
	resp, err := http.Get(archNewsURL)
	if err != nil {
		return nil, fmt.Errorf("request for newsfeed failed: %w", err)
	}
	defer closeBody(resp)

	slog.Debug("reading news")
	newsData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read newsfeed: %w", err)
	}

	return newsData, err
}

func decodeRss(newsData []byte) (*Rss, error) {
	buf := bytes.NewBuffer(newsData)

	rss := new(Rss)
	decoded := xml.NewDecoder(buf)

	slog.Debug("decoding news data")
	err := decoded.Decode(rss)
	if err != nil {
		return nil, fmt.Errorf("could not decode newsfeed: %w", err)
	}

	return rss, err
}

func closeBody(resp *http.Response) {
	slog.Debug("closing body")
	err := resp.Body.Close()
	if err != nil {
		panic(fmt.Errorf("failed to close http request body: %w", err))
	}
}

func loadSeenItemsFromFile(filename string) ([]string, error) {
	var seenBeforeItems []string
	slog.Debug("checking if seen items exist")
	_, err := os.Stat(filename)
	if err != nil {
		// Fail if it's any error other than ErrNotExist
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("checking if seen_items.json exists failed: %w", err)
		}

		// First run!
		slog.Debug("seen items does not exist: first run")

		return seenBeforeItems, nil
	}

	slog.Debug("seen items exist, reading seen items")
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read seen items.json: %w", err)
	}
	slog.Debug("unmarshalling seen items")
	err = json.Unmarshal(jsonData, &seenBeforeItems)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal seen_items.json: %w", err)
	}

	return seenBeforeItems, nil
}

func saveSeenItemsToFile(seenAfterItems []string, filename string) error {
	slog.Debug("marshalling seen items")
	jsonString, err := json.Marshal(seenAfterItems)
	if err != nil {
		return fmt.Errorf("could not marshal seenAfterItems: %w", err)
	}

	slog.Debug("saving seen items")
	err = os.WriteFile(filename, jsonString, writeFileMode)
	if err != nil {
		return fmt.Errorf("could not save the seen items: %w", err)
	}

	return nil
}
