// examples/picture/picsum.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	tea "charm.land/bubbletea/v2"
)

// picsumItem is one entry from picsum.photos /v2/list.
type picsumItem struct {
	ID          string `json:"id"`
	Author      string `json:"author"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
}

// picsumListLoadedMsg arrives when the catalog has been loaded (or failed).
type picsumListLoadedMsg struct {
	items []picsumItem
	err   error
}

// fetchListCmd returns a tea.Cmd that fetches the first `pages` pages of
// /v2/list at the given limit, concatenated into one slice.
func fetchListCmd(pages, limit int) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 15 * time.Second}
		var all []picsumItem
		for p := 1; p <= pages; p++ {
			url := fmt.Sprintf("https://picsum.photos/v2/list?page=%d&limit=%d", p, limit)
			resp, err := client.Get(url)
			if err != nil {
				return picsumListLoadedMsg{err: fmt.Errorf("fetch page %d: %w", p, err)}
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				resp.Body.Close()
				return picsumListLoadedMsg{err: fmt.Errorf("page %d: %s", p, resp.Status)}
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return picsumListLoadedMsg{err: fmt.Errorf("read page %d: %w", p, err)}
			}
			var batch []picsumItem
			if err := json.Unmarshal(body, &batch); err != nil {
				return picsumListLoadedMsg{err: fmt.Errorf("parse page %d: %w", p, err)}
			}
			if len(batch) == 0 {
				break
			}
			all = append(all, batch...)
		}
		if len(all) == 0 {
			return picsumListLoadedMsg{err: errors.New("empty catalog")}
		}
		return picsumListLoadedMsg{items: all}
	}
}

// imageURL builds the "small version" URL for an item.
func imageURL(id string) string {
	return fmt.Sprintf("https://picsum.photos/id/%s/400/300", id)
}

// findIndex returns the index of the item with the given ID, or -1.
func findIndex(items []picsumItem, id string) int {
	for i, it := range items {
		if it.ID == id {
			return i
		}
	}
	return -1
}
