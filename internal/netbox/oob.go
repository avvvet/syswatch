package netbox

import (
	"encoding/json"
	"fmt"
)

// SyncOOBAddress sets the out-of-band management IP on a device.
// This stores the iDRAC/iLO IP in NetBox device's oob_ip field.
func (c *Client) SyncOOBAddress(deviceID int, ipWithPrefix string) error {
	if ipWithPrefix == "" {
		return nil
	}

	// Check if OOB IP already set on this device
	var device struct {
		OOBIP *struct {
			ID      int    `json:"id"`
			Address string `json:"address"`
		} `json:"oob_ip"`
	}
	if err := c.get(fmt.Sprintf("/api/dcim/devices/%d/", deviceID), &device); err != nil {
		return err
	}

	if device.OOBIP != nil && device.OOBIP.Address == ipWithPrefix {
		c.log.Debug().Str("ip", ipWithPrefix).Msg("OOB IP unchanged")
		return nil
	}

	// Create IP in IPAM with role=oob
	var ipResult idResponse
	err := c.post("/api/ipam/ip-addresses/", map[string]interface{}{
		"address": ipWithPrefix,
		"status":  "active",
		"role":    "oob",
	}, &ipResult)

	if err != nil {
		// IP may already exist — find it
		var list listResponse
		if listErr := c.get(fmt.Sprintf(
			"/api/ipam/ip-addresses/?address=%s", ipWithPrefix,
		), &list); listErr != nil {
			return listErr
		}
		if list.Count == 0 {
			return fmt.Errorf("could not create or find OOB IP %s", ipWithPrefix)
		}
		var existing idResponse
		if err := json.Unmarshal(list.Results[0], &existing); err != nil {
			return err
		}
		ipResult.ID = existing.ID
	}

	// Assign to device as oob_ip
	var result idResponse
	return c.patch(fmt.Sprintf("/api/dcim/devices/%d/", deviceID),
		map[string]interface{}{
			"oob_ip": map[string]int{"id": ipResult.ID},
		}, &result)
}
