package netbox

import (
	"encoding/json"
	"fmt"
)

type ipAddressPayload struct {
	Address            string `json:"address"`
	AssignedObjectType string `json:"assigned_object_type"`
	AssignedObjectID   int    `json:"assigned_object_id"`
	Status             string `json:"status"`
}

type ipAddressResult struct {
	ID      int    `json:"id"`
	Address string `json:"address"`
}

// SyncIPAddress creates or updates an IP address assigned to an interface.
func (c *Client) SyncIPAddress(interfaceID int, ipWithPrefix string) error {
	if ipWithPrefix == "" {
		return nil
	}

	// Check if IP already assigned to this interface
	existing, err := c.findIPForInterface(interfaceID)
	if err != nil {
		return err
	}

	if existing != nil {
		if existing.Address == ipWithPrefix {
			c.log.Debug().Str("ip", ipWithPrefix).Msg("IP address unchanged")
			return nil
		}

		// IP changed — update it
		var result idResponse
		return c.patch(fmt.Sprintf("/api/ipam/ip-addresses/%d/", existing.ID),
			ipAddressPayload{
				Address:            ipWithPrefix,
				AssignedObjectType: "dcim.interface",
				AssignedObjectID:   interfaceID,
				Status:             "active",
			}, &result)
	}

	// Create new IP
	var created idResponse
	err = c.post("/api/ipam/ip-addresses/", ipAddressPayload{
		Address:            ipWithPrefix,
		AssignedObjectType: "dcim.interface",
		AssignedObjectID:   interfaceID,
		Status:             "active",
	}, &created)
	if err != nil {
		return fmt.Errorf("creating IP %s: %w", ipWithPrefix, err)
	}

	c.log.Info().Str("ip", ipWithPrefix).Int("id", created.ID).Msg("IP address created")
	return nil
}

func (c *Client) findIPForInterface(interfaceID int) (*ipAddressResult, error) {
	var list listResponse
	if err := c.get(fmt.Sprintf("/api/ipam/ip-addresses/?interface_id=%d", interfaceID), &list); err != nil {
		return nil, err
	}

	if list.Count == 0 {
		return nil, nil
	}

	var ip ipAddressResult
	if err := json.Unmarshal(list.Results[0], &ip); err != nil {
		return nil, err
	}

	return &ip, nil
}
