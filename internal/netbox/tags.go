package netbox

import (
	"fmt"
)

type tagPayload struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

// EnsureTag creates a tag if it does not exist.
func (c *Client) EnsureTag(name, color string) error {
	slug := generateSlug(name)

	var list listResponse
	if err := c.get(fmt.Sprintf("/api/extras/tags/?slug=%s", slug), &list); err != nil {
		return fmt.Errorf("fetching tag %q: %w", name, err)
	}

	if list.Count > 0 {
		c.log.Debug().Str("tag", name).Msg("tag already exists")
		return nil
	}

	var created idResponse
	err := c.post("/api/extras/tags/", tagPayload{
		Name:  name,
		Slug:  slug,
		Color: color,
	}, &created)
	if err != nil {
		return fmt.Errorf("creating tag %q: %w", name, err)
	}

	c.log.Info().Str("tag", name).Int("id", created.ID).Msg("tag created")
	return nil
}

// EnsureRequiredTags creates all tags syswatch needs.
//
// Tag meanings:
//
//	syswatch                        → device managed by syswatch
//	identified-by-motherboard-serial → no system serial, used motherboard serial (low priority)
//	identified-by-mac                → no serial, used MAC address (medium priority)
//	identified-by-machine-id         → no serial or MAC, used machine-id (high priority)
func (c *Client) EnsureRequiredTags() error {
	tags := []struct {
		name  string
		color string
	}{
		{"syswatch", "00bcd4"},                         // cyan
		{"identified-by-motherboard-serial", "ffc107"}, // amber  — low priority
		{"identified-by-mac", "ff9800"},                // orange — medium priority
		{"identified-by-machine-id", "f44336"},         // red    — high priority
	}

	for _, t := range tags {
		if err := c.EnsureTag(t.name, t.color); err != nil {
			return err
		}
	}

	return nil
}
