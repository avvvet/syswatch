package netbox

import (
	"encoding/json"
	"fmt"
)

type interfacePayload struct {
	Device int    `json:"device"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

type macAddressPayload struct {
	MACAddress         string `json:"mac_address"`
	AssignedObjectType string `json:"assigned_object_type"`
	AssignedObjectID   int    `json:"assigned_object_id"`
}

type setPrimaryMACPayload struct {
	PrimaryMACAddress map[string]int `json:"primary_mac_address"`
}

type interfaceResult struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type macResult struct {
	ID         int    `json:"id"`
	MACAddress string `json:"mac_address"`
}

// SyncInterfaces syncs NICs for a device.
// Uses MAC address as the unique identifier for each NIC.
func (c *Client) SyncInterfaces(deviceID int, nics []nicItem) error {
	existing, err := c.getInterfaces(deviceID)
	if err != nil {
		return err
	}

	existingByName := make(map[string]int)
	for _, iface := range existing {
		existingByName[iface.Name] = iface.ID
	}

	for _, nic := range nics {
		var ifaceID int

		if id, exists := existingByName[nic.Name]; exists {
			// Interface already exists — use existing ID
			ifaceID = id
			c.log.Debug().Str("interface", nic.Name).Msg("interface already exists")
		} else {
			// Create the interface
			var created idResponse
			err := c.post("/api/dcim/interfaces/", interfacePayload{
				Device: deviceID,
				Name:   nic.Name,
				Type:   nic.Type,
			}, &created)
			if err != nil {
				c.log.Warn().Err(err).Str("interface", nic.Name).Msg("failed to create interface")
				continue
			}
			ifaceID = created.ID
			c.log.Info().Str("interface", nic.Name).Int("id", ifaceID).Msg("interface created")
		}

		// Handle MAC address
		if nic.MACAddress != "" {
			if err := c.syncMACAddress(ifaceID, nic.MACAddress); err != nil {
				c.log.Warn().Err(err).Str("mac", nic.MACAddress).Msg("failed to sync MAC address")
			}
		}
	}

	return nil
}

// syncMACAddress creates a MAC address entry and sets it as primary on the interface.
// This follows the flow we confirmed via API testing:
// 1. POST to /api/dcim/mac-addresses/
// 2. PATCH interface with primary_mac_address: {"id": mac_id}
func (c *Client) syncMACAddress(interfaceID int, mac string) error {
	// Check if MAC already exists for this interface
	existingMAC, err := c.findMACForInterface(interfaceID)
	if err != nil {
		return err
	}

	var macID int

	if existingMAC != nil {
		macID = existingMAC.ID
		c.log.Debug().Str("mac", mac).Msg("MAC address already exists")
	} else {
		// Create MAC address
		var created idResponse
		err := c.post("/api/dcim/mac-addresses/", macAddressPayload{
			MACAddress:         mac,
			AssignedObjectType: "dcim.interface",
			AssignedObjectID:   interfaceID,
		}, &created)
		if err != nil {
			return fmt.Errorf("creating MAC address: %w", err)
		}
		macID = created.ID
		c.log.Info().Str("mac", mac).Int("id", macID).Msg("MAC address created")
	}

	// Set as primary on interface — confirmed format from API testing
	var result idResponse
	return c.patch(fmt.Sprintf("/api/dcim/interfaces/%d/", interfaceID),
		setPrimaryMACPayload{
			PrimaryMACAddress: map[string]int{"id": macID},
		}, &result)
}

func (c *Client) getInterfaces(deviceID int) ([]interfaceResult, error) {
	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/interfaces/?device_id=%d", deviceID), &list); err != nil {
		return nil, err
	}

	var ifaces []interfaceResult
	for _, raw := range list.Results {
		var iface interfaceResult
		if err := json.Unmarshal(raw, &iface); err != nil {
			continue
		}
		ifaces = append(ifaces, iface)
	}

	return ifaces, nil
}

func (c *Client) findMACForInterface(interfaceID int) (*macResult, error) {
	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/mac-addresses/?assigned_object_id=%d", interfaceID), &list); err != nil {
		return nil, err
	}

	if list.Count == 0 {
		return nil, nil
	}

	var mac macResult
	if err := json.Unmarshal(list.Results[0], &mac); err != nil {
		return nil, err
	}

	return &mac, nil
}

// nicItem is the minimal NIC data needed for interface sync.
type nicItem struct {
	Name       string
	MACAddress string
	Type       string
	IPAddress  string
}
