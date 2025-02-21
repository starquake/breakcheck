package checker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/starquake/breakcheck/feed"
)

const (
	httpRequestTimeout = 30 * time.Second
)

var (
	errGettingFeedHTTPStatusError = errors.New("error getting feed: http status error")
	errLastBuildDateIsEmpty       = errors.New("last build date is empty")
)

type Response struct {
	Changed            bool
	HeaderLastModified string
	FeedLastModified   string
}

func Check(ctx context.Context, archNewsURL string, headerLastModified string, feedLastBuildDate string) (*Response, error) {
	reqCtx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
	defer cancel()

	r := &Response{}

	req, requestErr := http.NewRequestWithContext(reqCtx, http.MethodGet, archNewsURL, nil)
	if requestErr != nil {
		return nil, requestErr
	}
	req.Header.Add("User-Agent", "breakcheck")
	req.Header.Add("Accept", "application/rss+xml")
	req.Header.Add("If-Modified-Since", headerLastModified)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	if resp.StatusCode == http.StatusNotModified {
		slog.Debug("feed has not been modified",
			"headerLastModified", headerLastModified,
			"r.headerLastModified", r.HeaderLastModified,
		)

		return r, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", errGettingFeedHTTPStatusError, resp.Status)
	}

	r.Changed = true
	r.HeaderLastModified = resp.Header.Get("Last-Modified")

	newsData, readError := io.ReadAll(resp.Body)
	if readError != nil {
		return nil, readError
	}

	lastBuildDate, decodeErr := feed.LastBuildDate(newsData)
	if decodeErr != nil {
		return nil, fmt.Errorf("error getting last build date: %w", decodeErr)
	}

	if lastBuildDate == "" {
		return nil, errLastBuildDateIsEmpty
	}

	r.FeedLastModified = lastBuildDate

	if lastBuildDate == feedLastBuildDate {
		r.Changed = false

		return r, nil
	}

	return r, nil
}

func closeBody(resp *http.Response) {
	slog.Debug("closing body")
	err := resp.Body.Close()
	if err != nil {
		panic(fmt.Errorf("failed to close http Check body: %w", err))
	}
}
