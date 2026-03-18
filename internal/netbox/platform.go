package netbox

import (
	"encoding/json"
	"fmt"
)

type platformPayload struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetOrCreatePlatform returns the ID of a platform (OS),
// creating it if it does not exist.
// Searches by slug to avoid case mismatch issues.
func (c *Client) GetOrCreatePlatform(name, slug string) (int, error) {
	if name == "" {
		return 0, nil
	}

	// Always search by slug — slug is always lowercase
	// avoids case sensitivity issues with name search
	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/platforms/?slug=%s", slug), &list); err != nil {
		return 0, fmt.Errorf("fetching platform %q: %w", name, err)
	}

	if list.Count > 0 {
		var platform idResponse
		if err := json.Unmarshal(list.Results[0], &platform); err != nil {
			return 0, err
		}
		c.log.Debug().Str("platform", name).Int("id", platform.ID).Msg("platform found")
		return platform.ID, nil
	}

	var created idResponse
	err := c.post("/api/dcim/platforms/", platformPayload{
		Name: name,
		Slug: slug,
	}, &created)
	if err != nil {
		return 0, fmt.Errorf("creating platform %q: %w", name, err)
	}

	c.log.Info().Str("platform", name).Int("id", created.ID).Msg("platform created")
	return created.ID, nil
}
