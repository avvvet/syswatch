package netbox

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type deviceTypePayload struct {
	Manufacturer int    `json:"manufacturer"`
	Model        string `json:"model"`
	Slug         string `json:"slug"`
	UHeight      int    `json:"u_height,omitempty"`
}

// GetOrCreateDeviceType returns the ID of a device type,
// creating it if it does not exist.
// Searches by slug to avoid duplicate slug errors.
func (c *Client) GetOrCreateDeviceType(manufacturerID int, model string, uHeight int) (int, error) {
	slug := generateSlug(model)

	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/device-types/?slug=%s&model=%s", slug, url.QueryEscape(model)), &list); err != nil {
		return 0, fmt.Errorf("fetching device type %q: %w", model, err)
	}

	if list.Count > 0 {
		var dt idResponse
		if err := json.Unmarshal(list.Results[0], &dt); err != nil {
			return 0, err
		}
		c.log.Debug().Str("model", model).Int("id", dt.ID).Msg("device type found")
		return dt.ID, nil
	}

	height := uHeight
	if height == 0 {
		height = 1
	}

	var created idResponse
	err := c.post("/api/dcim/device-types/", deviceTypePayload{
		Manufacturer: manufacturerID,
		Model:        model,
		Slug:         slug,
		UHeight:      height,
	}, &created)
	if err != nil {
		return 0, fmt.Errorf("creating device type %q: %w", model, err)
	}

	c.log.Info().Str("model", model).Int("id", created.ID).Msg("device type created")
	return created.ID, nil
}
