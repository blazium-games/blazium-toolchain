package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/blazium-games/blazium-toolchain/internal/report"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

// GetJSON GETs url and decodes a JSON object into dest.
func (h HTTP) GetJSON(ctx context.Context, url string, dest any) error {
	if err := settings.CheckFetchURL(url); err != nil {
		return err
	}
	c := h.Client
	if c == nil {
		c = &http.Client{Timeout: settings.Current().JSON()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", settings.Current().UserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return report.Fail(fmt.Sprintf("GET %s: HTTP %s", url, resp.Status), "check the catalog URL in blazium-toolchain.yml")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return report.Fail("decode "+url+": "+err.Error(), "the catalog JSON was not an object we understand")
	}
	return nil
}
