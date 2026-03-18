package netbox

import (
	"encoding/json"
	"fmt"
)

// GetSiteID returns the ID of a site by name.
// Sites are pre-created by humans — we never create them automatically.
// Returns an error if the site does not exist.
func (c *Client) GetSiteID(name string) (int, error) {
	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/sites/?name=%s", name), &list); err != nil {
		return 0, fmt.Errorf("fetching site %q: %w", name, err)
	}

	if list.Count == 0 {
		return 0, fmt.Errorf("site %q not found in NetBox — please create it manually", name)
	}

	var site idResponse
	if err := json.Unmarshal(list.Results[0], &site); err != nil {
		return 0, fmt.Errorf("parsing site response: %w", err)
	}

	c.log.Debug().Str("site", name).Int("id", site.ID).Msg("site found")
	return site.ID, nil
}
