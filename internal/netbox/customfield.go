package netbox

import (
	"encoding/json"
	"fmt"
)

type customFieldPayload struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	ObjectTypes []string `json:"object_types"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
}

type customFieldResult struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// EnsureCustomFields creates all custom fields syswatch needs on Device.
// Called on startup — safe to call multiple times, skips existing fields.
func (c *Client) EnsureCustomFields() error {
	fields := []customFieldPayload{
		{
			Name:        "cpu_model",
			Label:       "CPU Model",
			Type:        "text",
			ObjectTypes: []string{"dcim.device"},
			Description: "CPU model name reported by SMBIOS",
		},
		{
			Name:        "cpu_cores",
			Label:       "CPU Cores",
			Type:        "integer",
			ObjectTypes: []string{"dcim.device"},
			Description: "Total CPU cores across all sockets",
		},
		{
			Name:        "ram_gb",
			Label:       "RAM (GB)",
			Type:        "integer",
			ObjectTypes: []string{"dcim.device"},
			Description: "Total installed RAM in gigabytes",
		},
		{
			Name:        "bios_version",
			Label:       "BIOS Version",
			Type:        "text",
			ObjectTypes: []string{"dcim.device"},
			Description: "BIOS/firmware version reported by SMBIOS",
		},
		{
			Name:        "kernel",
			Label:       "Kernel",
			Type:        "text",
			ObjectTypes: []string{"dcim.device"},
			Description: "Linux kernel version",
		},
		{
			Name:        "identifier_source",
			Label:       "Identifier Source",
			Type:        "text",
			ObjectTypes: []string{"dcim.device"},
			Description: "How syswatch identified this device: smbios-serial, motherboard-serial, mac-address, machine-id",
		},
	}

	for _, field := range fields {
		if err := c.ensureCustomField(field); err != nil {
			return fmt.Errorf("ensuring custom field %q: %w", field.Name, err)
		}
	}

	return nil
}

// ensureCustomField creates a custom field if it does not already exist.
func (c *Client) ensureCustomField(field customFieldPayload) error {
	// Check if already exists
	var list struct {
		Count   int               `json:"count"`
		Results []json.RawMessage `json:"results"`
	}
	if err := c.get(fmt.Sprintf("/api/extras/custom-fields/?name=%s", field.Name), &list); err != nil {
		return fmt.Errorf("checking custom field: %w", err)
	}

	if list.Count > 0 {
		c.log.Debug().Str("field", field.Name).Msg("custom field already exists")
		return nil
	}

	// Create it
	var result customFieldResult
	if err := c.post("/api/extras/custom-fields/", field, &result); err != nil {
		return fmt.Errorf("creating custom field: %w", err)
	}

	c.log.Info().
		Str("field", field.Name).
		Int("id", result.ID).
		Msg("custom field created")

	return nil
}
