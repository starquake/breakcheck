package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/starquake/breakcheck/check"
	"github.com/starquake/breakcheck/feed"
	"github.com/starquake/breakcheck/store"
)

const (
	archNewsURL = "https://archlinux.org/feeds/news/"
	storeFile   = "breakcheck.json"
)

var (
	ExitCodeSuccess = 0
	ExitCodeNoNews  = 1
	ExitCodeOther   = 100
)

func run(ctx context.Context) (bool, error) {
	slog.Debug("about to load store")
	breakcheckStore := &store.Store{}

	err := breakcheckStore.LoadStoreFromFile(storeFile)
	if err != nil {
		return false, fmt.Errorf("error loading store from file: %w", err)
	}

	resp, err := check.Request(ctx, breakcheckStore.HeaderLastModified, archNewsURL)
	if err != nil {
		return false, fmt.Errorf("error executing Request: %w", err)
	}

	if resp.Status == http.StatusNotModified {
		// No change, so return false with no error
		return false, nil
	}

	breakcheckStore.HeaderLastModified = resp.HeaderLastModified
	err = saveToStore(breakcheckStore)
	if err != nil {
		return false, fmt.Errorf("error saving store to file after setting HeaderLastModified: %w", err)
	}

	lastBuildDate, decodeErr := feed.LastBuildDate(resp.ResponseData)
	if decodeErr != nil {
		return false, fmt.Errorf("error getting last build date: %w", decodeErr)
	}

	if lastBuildDate == "" {
		return false, nil
	}

	// We have a response, so there could be check
	// but let's check if the feed LastBuild Date is newer just to be sure

	if lastBuildDate == breakcheckStore.FeedLastBuildDate {
		// No check, no error
		return false, nil
	}

	// Update that we've seen a check
	breakcheckStore.FeedLastBuildDate = lastBuildDate
	err = saveToStore(breakcheckStore)
	if err != nil {
		return false, fmt.Errorf("error saving store to file after update FeedLastBuildDate: %w", err)
	}

	fmt.Printf("News has been updated.\n")
	fmt.Printf("Make sure you have read the check items and restart the upgrade to complete.\n")
	fmt.Printf("\n")

	return true, nil
}

func saveToStore(breakcheckStore *store.Store) error {
	err := breakcheckStore.SaveStoreToFile(storeFile)
	if err != nil {
		return fmt.Errorf("error saving store to file: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	news, err := run(ctx)
	if err != nil {
		_, writeErr := fmt.Fprintf(os.Stderr, "%v\n", err)
		if writeErr != nil {
			panic(writeErr)
		}
		os.Exit(ExitCodeOther)
	}
	if news {
		os.Exit(ExitCodeNoNews)
	}
	os.Exit(ExitCodeSuccess)
}
