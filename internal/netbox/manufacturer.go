package netbox

import (
	"encoding/json"
	"fmt"
)

type manufacturerPayload struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetOrCreateManufacturer returns the ID of a manufacturer,
// creating it if it does not exist.
// Searches by slug to avoid duplicate slug errors.
func (c *Client) GetOrCreateManufacturer(name string) (int, error) {
	slug := generateSlug(name)

	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/manufacturers/?slug=%s", slug), &list); err != nil {
		return 0, fmt.Errorf("fetching manufacturer %q: %w", name, err)
	}

	if list.Count > 0 {
		var mfr idResponse
		if err := json.Unmarshal(list.Results[0], &mfr); err != nil {
			return 0, err
		}
		c.log.Debug().Str("manufacturer", name).Int("id", mfr.ID).Msg("manufacturer found")
		return mfr.ID, nil
	}

	var created idResponse
	err := c.post("/api/dcim/manufacturers/", manufacturerPayload{
		Name: name,
		Slug: slug,
	}, &created)
	if err != nil {
		return 0, fmt.Errorf("creating manufacturer %q: %w", name, err)
	}

	c.log.Info().Str("manufacturer", name).Int("id", created.ID).Msg("manufacturer created")
	return created.ID, nil
}
