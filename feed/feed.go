package feed

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

type Rss struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	LastBuildDate string `xml:"lastBuildDate"`
}

func LastBuildDate(newsData []byte) (string, error) {
	buf := bytes.NewBuffer(newsData)
	rss := new(Rss)
	decoder := xml.NewDecoder(buf)

	decodeError := decoder.Decode(rss)
	if decodeError != nil {
		return "", fmt.Errorf("could not feed newsfeed: %w", decodeError)
	}

	return rss.Channel.LastBuildDate, nil
}
