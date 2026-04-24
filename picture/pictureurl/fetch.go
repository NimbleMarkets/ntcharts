package pictureurl

import (
	"bytes"
	"errors"
	"image"
	"io"
	"net/http"

	tea "charm.land/bubbletea/v2"
)

type imageLoadedMsg struct {
	url string
	img image.Image
	err error
}

func fetchCmd(client *http.Client, url string, maxSize int64) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.Get(url)
		if err != nil {
			return imageLoadedMsg{url: url, err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return imageLoadedMsg{url: url, err: errors.New(resp.Status)}
		}

		data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
		if err != nil {
			return imageLoadedMsg{url: url, err: err}
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return imageLoadedMsg{url: url, err: err}
		}

		return imageLoadedMsg{url: url, img: img}
	}
}
