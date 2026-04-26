package pictureurl

import (
	"bytes"
	"errors"
	"image"
	"io"
	"net/http"

	tea "charm.land/bubbletea/v2"
)

type ImageLoadedMsg struct {
	URL string
	Img image.Image
	Err error
}

func fetchCmd(client *http.Client, url string, maxSize int64) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.Get(url)
		if err != nil {
			return ImageLoadedMsg{URL: url, Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return ImageLoadedMsg{URL: url, Err: errors.New(resp.Status)}
		}

		data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
		if err != nil {
			return ImageLoadedMsg{URL: url, Err: err}
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return ImageLoadedMsg{URL: url, Err: err}
		}

		return ImageLoadedMsg{URL: url, Img: img}
	}
}
