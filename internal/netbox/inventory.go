package netbox

import (
	"encoding/json"
	"fmt"
)

type inventoryItemPayload struct {
	Device       int    `json:"device"`
	Name         string `json:"name"`
	Manufacturer *int   `json:"manufacturer,omitempty"`
	PartID       string `json:"part_id,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Description  string `json:"description,omitempty"`
	Discovered   bool   `json:"discovered"`
}

type inventoryItemResult struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SyncInventoryItems syncs inventory items for a device.
// Items that exist are updated, new items are created.
// Items no longer present in hardware scan are deleted from NetBox.
func (c *Client) SyncInventoryItems(deviceID int, items []inventoryItemPayload) error {
	// Get existing items for this device
	existing, err := c.getInventoryItems(deviceID)
	if err != nil {
		return err
	}

	// Build a map of existing items by name for quick lookup
	existingByName := make(map[string]int)
	for _, item := range existing {
		existingByName[item.Name] = item.ID
	}

	// Track which names we are syncing
	syncedNames := make(map[string]bool)

	for _, item := range items {
		syncedNames[item.Name] = true

		if id, exists := existingByName[item.Name]; exists {
			// Update existing item
			if err := c.updateInventoryItem(id, item); err != nil {
				c.log.Warn().Err(err).Str("item", item.Name).Msg("failed to update inventory item")
			} else {
				c.log.Debug().Str("item", item.Name).Msg("inventory item updated")
			}
		} else {
			// Create new item
			if err := c.createInventoryItem(item); err != nil {
				c.log.Warn().Err(err).Str("item", item.Name).Msg("failed to create inventory item")
			} else {
				c.log.Info().Str("item", item.Name).Msg("inventory item created")
			}
		}
	}

	// Delete items in NetBox that are no longer found in hardware scan
	// This reflects actual hardware changes (RAM removed, disk replaced etc)
	for name, id := range existingByName {
		if !syncedNames[name] {
			if err := c.deleteInventoryItem(id); err != nil {
				c.log.Warn().Err(err).Str("item", name).Msg("failed to delete removed inventory item")
			} else {
				c.log.Info().
					Str("item", name).
					Int("device_id", deviceID).
					Msg("inventory item removed from hardware — deleted from NetBox")
			}
		}
	}

	return nil
}

func (c *Client) getInventoryItems(deviceID int) ([]inventoryItemResult, error) {
	var list listResponse
	if err := c.get(fmt.Sprintf("/api/dcim/inventory-items/?device_id=%d&limit=100", deviceID), &list); err != nil {
		return nil, err
	}

	var items []inventoryItemResult
	for _, raw := range list.Results {
		var item inventoryItemResult
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func (c *Client) createInventoryItem(item inventoryItemPayload) error {
	var result idResponse
	return c.post("/api/dcim/inventory-items/", item, &result)
}

func (c *Client) updateInventoryItem(id int, item inventoryItemPayload) error {
	var result idResponse
	return c.patch(fmt.Sprintf("/api/dcim/inventory-items/%d/", id), item, &result)
}

// BuildInventoryItems converts hardware data into NetBox inventory payloads.
func BuildInventoryItems(
	deviceID int,
	cpuModel string,
	cpuCores, cpuSockets int,
	memory interface{ GetModules() []interface{} },
	disks []diskItem,
	manufacturerIDs map[string]int,
) []inventoryItemPayload {
	var items []inventoryItemPayload

	// CPU entries — one per socket
	for i := 0; i < cpuSockets; i++ {
		items = append(items, inventoryItemPayload{
			Device:      deviceID,
			Name:        fmt.Sprintf("CPU %d", i),
			Description: fmt.Sprintf("%s (%d cores)", cpuModel, cpuCores),
			Discovered:  true,
		})
	}

	return items
}

// diskItem is a minimal interface for disk data needed here.
type diskItem struct {
	Name         string
	Manufacturer string
	Model        string
	Serial       string
	SizeGB       int
	Type         string
}

func (c *Client) deleteInventoryItem(id int) error {
	return c.delete(fmt.Sprintf("/api/dcim/inventory-items/%d/", id))
}
