package pictureurl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type ImageLoadedMsg struct {
	modelID uint64
	seq     uint64
	URL     string
	Img     image.Image
	Err     error
}

var (
	ErrImageTooLarge           = errors.New("image response exceeds max size")
	ErrImageDimensionsTooLarge = errors.New("image dimensions exceed max pixels")
	ErrUnexpectedContentType   = errors.New("unexpected Content-Type")
)

func fetchCmd(modelID, seq uint64, client *http.Client, userAgent, url string, maxSize int64, maxPixels int) tea.Cmd {
	mkErr := func(err error) tea.Msg {
		return ImageLoadedMsg{modelID: modelID, seq: seq, URL: url, Err: err}
	}
	return func() tea.Msg {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return mkErr(err)
		}
		if userAgent != "" {
			req.Header.Set("User-Agent", userAgent)
		}
		req.Header.Set("Accept", "image/*")

		resp, err := client.Do(req)
		if err != nil {
			return mkErr(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return mkErr(errors.New(resp.Status))
		}

		if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "image/") {
			return mkErr(fmt.Errorf("%w: %q", ErrUnexpectedContentType, ct))
		}

		if resp.ContentLength > maxSize {
			return mkErr(fmt.Errorf("%w: %d > %d bytes", ErrImageTooLarge, resp.ContentLength, maxSize))
		}

		data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
		if err != nil {
			return mkErr(err)
		}
		if int64(len(data)) > maxSize {
			return mkErr(fmt.Errorf("%w: > %d bytes", ErrImageTooLarge, maxSize))
		}

		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return mkErr(err)
		}
		if maxPixels > 0 && cfg.Width > 0 && cfg.Height > maxPixels/cfg.Width {
			return mkErr(fmt.Errorf("%w: %dx%d > %d pixels", ErrImageDimensionsTooLarge, cfg.Width, cfg.Height, maxPixels))
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return mkErr(err)
		}

		return ImageLoadedMsg{modelID: modelID, seq: seq, URL: url, Img: img}
	}
}
