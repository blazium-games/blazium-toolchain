package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GetJSON GETs url and decodes a JSON object into dest.
func (h HTTP) GetJSON(ctx context.Context, url string, dest any) error {
	if url == "" {
		return fmt.Errorf("empty url")
	}
	c := h.Client
	if c == nil {
		c = &http.Client{Timeout: 60 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "blazium-toolchain/0.1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}
