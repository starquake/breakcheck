package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/starquake/breakcheck/checker"
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

	resp, err := checker.Check(ctx, archNewsURL, breakcheckStore.HeaderLastModified, breakcheckStore.FeedLastBuildDate)
	if err != nil {
		return false, fmt.Errorf("error executing Check: %w", err)
	}

	if !resp.Changed {
		return resp.Changed, nil
	}

	breakcheckStore.HeaderLastModified = resp.HeaderLastModified
	breakcheckStore.FeedLastBuildDate = resp.FeedLastModified

	err = breakcheckStore.SaveStoreToFile(storeFile)
	if err != nil {
		return resp.Changed, fmt.Errorf("error saving store to file: %w", err)
	}

	fmt.Println("News has been updated.")
	fmt.Println("Make sure you have read the check items and restart the upgrade to complete.")
	fmt.Println("")

	return resp.Changed, nil
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
