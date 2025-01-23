package check

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	httpRequestTimeout = 30 * time.Second
)

type Response struct {
	Status             int
	HeaderLastModified string
	ResponseData       []byte
}

func Request(ctx context.Context, headerLastModified string, archNewsURL string) (*Response, error) {
	reqCtx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
	defer cancel()
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
		slog.Debug("check feed has not modified", "headerLastModified", headerLastModified)

		return &Response{
			Status:             resp.StatusCode,
			HeaderLastModified: resp.Header.Get("Last-Modified"),
		}, nil
	}

	newsData, readError := io.ReadAll(resp.Body)
	if readError != nil {
		return nil, readError
	}

	return &Response{
		Status:             resp.StatusCode,
		HeaderLastModified: resp.Header.Get("Last-Modified"),
		ResponseData:       newsData,
	}, nil
}

func closeBody(resp *http.Response) {
	slog.Debug("closing body")
	err := resp.Body.Close()
	if err != nil {
		panic(fmt.Errorf("failed to close http Request body: %w", err))
	}
}
