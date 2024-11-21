package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/starquake/breakcheck/store"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	archNewsURL        = "https://archlinux.org/feeds/news/"
	storeFile          = "/var/lib/breakcheck.json"
	httpRequestTimeout = 30 * time.Second
)

type ApplicationError struct {
	Code    int
	Message string
	Err     error
}

func (e *ApplicationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: (code %d): %v", e.Message, e.Code, e.Err)
	}

	return fmt.Sprintf("%s: (code %d)", e.Message, e.Code)
}

func (e *ApplicationError) Unwrap() error { return e.Err }

//nolint:mnd //These are note magic numbers, they are actual errorcodes
var (
	ErrLoadStoreFromFileFailedError = &ApplicationError{Code: 100, Message: "error loading store from file"}
	ErrCreatingRequestFailedError   = &ApplicationError{Code: 101, Message: "error creating request"}
	ErrRetrieveNewsDataFailedError  = &ApplicationError{Code: 102, Message: "retrieve news data failed"}
	ErrReadNewsDataFailedError      = &ApplicationError{Code: 103, Message: "read news data failed"}
	ErrDecodeNewsDataFailedError    = &ApplicationError{Code: 104, Message: "decode news data failed"}
	ErrSaveStoreToFieldError        = &ApplicationError{Code: 105, Message: "save store to field failed"}
)

type Rss struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	LastBuildDate string `xml:"lastBuildDate"`
}

func run(ctx context.Context) (bool, *ApplicationError) {
	slog.Debug("about to load store")
	breakcheckStore := &store.Store{}

	err := breakcheckStore.LoadStoreFromFile(storeFile)
	if err != nil {
		return false, &ApplicationError{
			Code:    ErrLoadStoreFromFileFailedError.Code,
			Message: ErrLoadStoreFromFileFailedError.Message,
			Err:     err,
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
	defer cancel()

	req, err := CreateRequest(reqCtx, breakcheckStore)
	if err != nil {
		return false, &ApplicationError{
			Code:    ErrCreatingRequestFailedError.Code,
			Message: ErrCreatingRequestFailedError.Message,
			Err:     err,
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, &ApplicationError{
			Code:    ErrRetrieveNewsDataFailedError.Code,
			Message: ErrRetrieveNewsDataFailedError.Message,
			Err:     err,
		}
	}
	defer closeBody(resp)

	if resp.StatusCode == http.StatusNotModified {
		slog.Debug("news feed has not modified", "headerLastModified", breakcheckStore.HeaderLastModified)

		return false, nil
	}

	breakcheckStore.HeaderLastModified = resp.Header.Get("Last-Modified")

	newsData, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, &ApplicationError{
			Code:    ErrReadNewsDataFailedError.Code,
			Message: ErrReadNewsDataFailedError.Message,
			Err:     err,
		}
	}

	slog.Debug("about to decode RSS")
	rss, err := decodeRss(newsData)
	if err != nil {
		return false, &ApplicationError{
			Code:    ErrDecodeNewsDataFailedError.Code,
			Message: ErrDecodeNewsDataFailedError.Message,
			Err:     err,
		}
	}

	if rss.Channel.LastBuildDate == breakcheckStore.FeedLastBuildDate {
		// No news, no error
		return false, nil
	}

	breakcheckStore.FeedLastBuildDate = rss.Channel.LastBuildDate

	err = breakcheckStore.SaveStoreToFile(storeFile)
	if err != nil {
		return false, &ApplicationError{
			Code:    ErrSaveStoreToFieldError.Code,
			Message: ErrSaveStoreToFieldError.Message,
			Err:     err,
		}
	}

	fmt.Printf("News has been updated.\n")
	fmt.Printf("Make sure you have read the news items and restart the upgrade to complete.\n")
	fmt.Printf("\n")

	return true, nil
}

func CreateRequest(ctx context.Context, store *store.Store) (*http.Request, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archNewsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "breakcheck")
	req.Header.Add("Accept", "application/rss+xml")
	req.Header.Add("If-Modified-Since", store.HeaderLastModified)

	return req, nil
}

func main() {
	ctx := context.Background()
	news, err := run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(err.Code)
	}
	if news {
		os.Exit(1)
	}
	os.Exit(0)
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
