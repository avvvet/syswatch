package netbox

import (
	"encoding/json"
	"fmt"
	"os"
)

type devicePayload struct {
	Name         string                 `json:"name"`
	DeviceType   int                    `json:"device_type"`
	Role         int                    `json:"role"`
	Site         int                    `json:"site"`
	Serial       string                 `json:"serial,omitempty"`
	Status       string                 `json:"status"`
	Platform     *int                   `json:"platform,omitempty"`
	Tags         []map[string]string    `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

type deviceUpdatePayload struct {
	Name         string                 `json:"name"`
	Serial       string                 `json:"serial,omitempty"`
	Platform     *int                   `json:"platform,omitempty"`
	Tags         []map[string]string    `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// GetOrCreateDevice returns the ID of a device, creating it if it does not exist.
// Kept for standalone mode compatibility.
func (c *Client) GetOrCreateDevice(
	identifierValue string,
	identifierSource string,
	hostname string,
	deviceTypeID, roleID, siteID int,
	customFields map[string]interface{},
	platformID *int,
	tags []map[string]string,
) (int, error) {
	id, found, err := c.FindDeviceBySerial(identifierValue)
	if err != nil {
		return 0, err
	}
	if found {
		return id, c.UpdateDevice(id, identifierValue, customFields, platformID, tags)
	}
	return c.CreateDevice(identifierValue, identifierSource, hostname, deviceTypeID, roleID, siteID, customFields, platformID, tags)
}

// CreateDevice creates a new device in NetBox.
// Device type is set here and never changed after.
func (c *Client) CreateDevice(
	identifierValue string,
	identifierSource string,
	hostname string,
	deviceTypeID, roleID, siteID int,
	customFields map[string]interface{},
	platformID *int,
	tags []map[string]string,
) (int, error) {
	h := hostname
	if h == "" {
		h, _ = os.Hostname()
	}
	name := fmt.Sprintf("%s-%s", h, identifierValue)

	var created idResponse
	err := c.post("/api/dcim/devices/", devicePayload{
		Name:         name,
		DeviceType:   deviceTypeID,
		Role:         roleID,
		Site:         siteID,
		Serial:       identifierValue,
		Status:       "active",
		Platform:     platformID,
		Tags:         tags,
		CustomFields: customFields,
	}, &created)
	if err != nil {
		return 0, fmt.Errorf("creating device %q: %w", name, err)
	}

	c.log.Info().Str("name", name).Int("id", created.ID).Msg("device created")
	return created.ID, nil
}

// UpdateDevice updates an existing device — never touches device type.
func (c *Client) UpdateDevice(
	id int,
	serial string,
	customFields map[string]interface{},
	platformID *int,
	tags []map[string]string,
) error {
	h, _ := os.Hostname()
	name := fmt.Sprintf("%s-%s", h, serial)
	return c.updateDevice(id, name, serial, customFields, platformID, tags)
}

// findDeviceBySerial searches for a device by serial number field.
func (c *Client) FindDeviceBySerial(serial string) (int, bool, error) {
	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/devices/?serial=%s", serial), &list); err != nil {
		return 0, false, fmt.Errorf("searching device by serial: %w", err)
	}

	if list.Count == 0 {
		return 0, false, nil
	}

	var device idResponse
	if err := json.Unmarshal(list.Results[0], &device); err != nil {
		return 0, false, err
	}

	c.log.Debug().Str("serial", serial).Int("id", device.ID).Msg("device found")
	return device.ID, true, nil
}

// updateDevice patches an existing device with latest fields.
// Name is always updated — binary is source of truth.
func (c *Client) updateDevice(
	id int,
	name string,
	serial string,
	customFields map[string]interface{},
	platformID *int,
	tags []map[string]string,
) error {
	payload := deviceUpdatePayload{
		Name:         name,
		Serial:       serial,
		Platform:     platformID,
		Tags:         tags,
		CustomFields: customFields,
	}

	var result idResponse
	err := c.patch(fmt.Sprintf("/api/dcim/devices/%d/", id), payload, &result)
	if err != nil {
		return fmt.Errorf("updating device %d: %w", id, err)
	}

	c.log.Debug().Int("id", id).Str("name", name).Msg("device updated")
	return nil
}
