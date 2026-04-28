package pictureurl

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"

	tea "charm.land/bubbletea/v2"
)

type ImageLoadedMsg struct {
	modelID uint64
	URL     string
	Img     image.Image
	Err     error
}

var (
	ErrImageTooLarge           = errors.New("image response exceeds max size")
	ErrImageDimensionsTooLarge = errors.New("image dimensions exceed max pixels")
)

func fetchCmd(modelID uint64, client *http.Client, url string, maxSize int64, maxPixels int) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.Get(url)
		if err != nil {
			return ImageLoadedMsg{modelID: modelID, URL: url, Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return ImageLoadedMsg{modelID: modelID, URL: url, Err: errors.New(resp.Status)}
		}

		if resp.ContentLength > maxSize {
			return ImageLoadedMsg{modelID: modelID, URL: url, Err: fmt.Errorf("%w: %d > %d bytes", ErrImageTooLarge, resp.ContentLength, maxSize)}
		}

		data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
		if err != nil {
			return ImageLoadedMsg{modelID: modelID, URL: url, Err: err}
		}
		if int64(len(data)) > maxSize {
			return ImageLoadedMsg{modelID: modelID, URL: url, Err: fmt.Errorf("%w: > %d bytes", ErrImageTooLarge, maxSize)}
		}

		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return ImageLoadedMsg{modelID: modelID, URL: url, Err: err}
		}
		if maxPixels > 0 && cfg.Width > 0 && cfg.Height > maxPixels/cfg.Width {
			return ImageLoadedMsg{modelID: modelID, URL: url, Err: fmt.Errorf("%w: %dx%d > %d pixels", ErrImageDimensionsTooLarge, cfg.Width, cfg.Height, maxPixels)}
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return ImageLoadedMsg{modelID: modelID, URL: url, Err: err}
		}

		return ImageLoadedMsg{modelID: modelID, URL: url, Img: img}
	}
}
