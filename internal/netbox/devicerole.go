package netbox

import (
	"encoding/json"
	"fmt"
)

type deviceRolePayload struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

// GetOrCreateDeviceRole returns the ID of a device role,
// creating it if it does not exist.
// Searches by slug first to avoid duplicate slug errors.
func (c *Client) GetOrCreateDeviceRole(name string) (int, error) {
	slug := generateSlug(name)

	// Search by slug — more reliable than name
	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/device-roles/?slug=%s", slug), &list); err != nil {
		return 0, fmt.Errorf("fetching device role %q: %w", name, err)
	}

	if list.Count > 0 {
		var role idResponse
		if err := json.Unmarshal(list.Results[0], &role); err != nil {
			return 0, err
		}
		c.log.Debug().Str("role", name).Int("id", role.ID).Msg("device role found")
		return role.ID, nil
	}

	// Create it
	var created idResponse
	err := c.post("/api/dcim/device-roles/", deviceRolePayload{
		Name:  name,
		Slug:  slug,
		Color: "0000ff",
	}, &created)
	if err != nil {
		return 0, fmt.Errorf("creating device role %q: %w", name, err)
	}

	c.log.Info().Str("role", name).Int("id", created.ID).Msg("device role created")
	return created.ID, nil
}
