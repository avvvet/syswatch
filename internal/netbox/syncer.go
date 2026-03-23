package netbox

import (
	"fmt"
	"strings"

	"github.com/avvvet/syswatch/internal/intelligence"
	"github.com/avvvet/syswatch/pkg/models"
)

// Syncer orchestrates the full hardware sync to NetBox.
type Syncer struct {
	client    *Client
	site      string
	role      string
	apiClient *intelligence.Client // nil in standalone mode
}

// NewSyncer creates a new Syncer in standalone mode.
func NewSyncer(client *Client, site, role string) *Syncer {
	return &Syncer{
		client: client,
		site:   site,
		role:   role,
	}
}

// NewSyncerWithAPI creates a new Syncer in API mode.
// All manufacturer, device type and platform resolution
// goes through the Central Intelligence API.
func NewSyncerWithAPI(client *Client, site, role string, apiClient *intelligence.Client) *Syncer {
	return &Syncer{
		client:    client,
		site:      site,
		role:      role,
		apiClient: apiClient,
	}
}

// isAPIMode returns true when running with Central Intelligence API.
func (s *Syncer) isAPIMode() bool {
	return s.apiClient != nil
}

// Sync pushes a complete hardware snapshot to NetBox.
func (s *Syncer) Sync(hw models.Hardware) error {
	log := s.client.log

	if hw.Identifier.Value == "" {
		return fmt.Errorf("cannot sync — no unique identifier resolved for this device")
	}

	log.Info().
		Str("identifier", hw.Identifier.Value).
		Str("identifier_source", hw.Identifier.Source).
		Str("mode", s.mode()).
		Msg("starting NetBox sync")

	// STEP 0 — Ensure required tags and custom fields exist
	if err := s.client.EnsureRequiredTags(); err != nil {
		return fmt.Errorf("step 0 tags: %w", err)
	}
	if err := s.client.EnsureCustomFields(); err != nil {
		return fmt.Errorf("step 0 custom fields: %w", err)
	}
	log.Info().Msg("step 0 tags and custom fields ok")

	// STEP 1 — Site
	siteID, err := s.client.GetSiteID(s.site)
	if err != nil {
		return fmt.Errorf("step 1 site: %w", err)
	}
	log.Info().Int("site_id", siteID).Msg("step 1 site ok")

	// STEP 2 — Check if device already exists
	// If it does we skip device type resolution entirely
	// Device type is set ONCE on creation and never updated
	existingDeviceID, deviceExists, err := s.client.FindDeviceBySerial(hw.Identifier.Value)
	if err != nil {
		return fmt.Errorf("step 2 device lookup: %w", err)
	}

	// STEP 3 — Manufacturer (always needed for context)
	mfrID, err := s.resolveManufacturer(hw.System.Manufacturer)
	if err != nil {
		return fmt.Errorf("step 3 manufacturer: %w", err)
	}
	log.Info().Int("manufacturer_id", mfrID).Msg("step 3 manufacturer ok")

	// STEP 4 — Device Role
	roleID, err := s.client.GetOrCreateDeviceRole(s.role)
	if err != nil {
		return fmt.Errorf("step 4 device role: %w", err)
	}
	log.Info().Int("role_id", roleID).Msg("step 4 device role ok")

	// STEP 5 — Device Type (only on first creation)
	var deviceTypeID int
	if !deviceExists {
		deviceTypeID, err = s.resolveDeviceType(hw.System.Model, hw.System.Manufacturer, hw.System.UHeight, mfrID)
		if err != nil {
			return fmt.Errorf("step 5 device type: %w", err)
		}
		log.Info().Int("device_type_id", deviceTypeID).Msg("step 5 device type ok")
	} else {
		log.Debug().Msg("step 5 device type skipped — device exists, respecting current type")
	}

	// STEP 6 — Platform (OS)
	var platformID *int
	if hw.OS.Name != "" {
		osName := fmt.Sprintf("%s %s", hw.OS.Name, hw.OS.Version)
		pid, err := s.resolvePlatform(osName, hw.OS.Slug)
		if err != nil {
			log.Warn().Err(err).Msg("platform sync failed — continuing")
		} else {
			platformID = &pid
			log.Info().Int("platform_id", pid).Msg("step 6 platform ok")
		}
	}

	// STEP 7 — Custom fields
	customFields := buildCustomFields(hw)

	// STEP 8 — Tags
	tags := buildTags(hw.Identifier.Source)

	// STEP 9 — Device
	var deviceID int
	if deviceExists {
		// UPDATE — never touch device type
		if err := s.client.UpdateDevice(existingDeviceID, hw.Identifier.Value, customFields, platformID, tags); err != nil {
			return fmt.Errorf("step 9 device update: %w", err)
		}
		deviceID = existingDeviceID
		log.Info().Int("device_id", deviceID).Msg("step 9 device updated")
	} else {
		// CREATE — set device type once
		deviceID, err = s.client.CreateDevice(
			hw.Identifier.Value,
			hw.Identifier.Source,
			"",
			deviceTypeID,
			roleID,
			siteID,
			customFields,
			platformID,
			tags,
		)
		if err != nil {
			return fmt.Errorf("step 9 device create: %w", err)
		}
		log.Info().Int("device_id", deviceID).Msg("step 9 device created")
	}

	// STEP 9 — Inventory Items
	inventoryItems := buildInventoryPayloads(deviceID, hw)
	if err := s.client.SyncInventoryItems(deviceID, inventoryItems); err != nil {
		log.Warn().Err(err).Msg("inventory sync failed — continuing")
	} else {
		log.Info().Int("item_count", len(inventoryItems)).Msg("step 9 inventory ok")
	}

	// STEP 10 — Interfaces and MACs
	nics := buildNICItems(hw.NICs)
	if err := s.client.SyncInterfaces(deviceID, nics); err != nil {
		log.Warn().Err(err).Msg("interface sync failed — continuing")
	} else {
		log.Info().Int("nic_count", len(nics)).Msg("step 10 interfaces ok")
	}

	log.Info().
		Int("device_id", deviceID).
		Str("identifier", hw.Identifier.Value).
		Msg("NetBox sync complete")

	return nil
}

// resolveManufacturer resolves manufacturer via API or standalone.
func (s *Syncer) resolveManufacturer(name string) (int, error) {
	if s.isAPIMode() {
		resp, err := s.apiClient.ResolveManufacturer(name)
		if err != nil {
			return 0, fmt.Errorf("API resolve manufacturer: %w", err)
		}
		s.client.log.Info().
			Str("raw", name).
			Str("canonical", resp.CanonicalName).
			Str("confidence", resp.Confidence).
			Msg("manufacturer resolved via API")
		return resp.NetBoxID, nil
	}

	// Standalone mode
	return s.client.GetOrCreateManufacturer(name)
}

// resolveDeviceType resolves device type via API or standalone.
func (s *Syncer) resolveDeviceType(model, manufacturer string, uHeight, mfrID int) (int, error) {
	if s.isAPIMode() {
		resp, err := s.apiClient.ResolveDeviceType(model, manufacturer)
		if err != nil {
			return 0, fmt.Errorf("API resolve device type: %w", err)
		}
		s.client.log.Info().
			Str("raw", model).
			Str("canonical", resp.CanonicalName).
			Str("confidence", resp.Confidence).
			Msg("device type resolved via API")
		return resp.NetBoxID, nil
	}

	// Standalone mode
	return s.client.GetOrCreateDeviceType(mfrID, model, uHeight)
}

// resolvePlatform resolves platform via API or standalone.
func (s *Syncer) resolvePlatform(name, slug string) (int, error) {
	if s.isAPIMode() {
		resp, err := s.apiClient.ResolvePlatform(name)
		if err != nil {
			return 0, fmt.Errorf("API resolve platform: %w", err)
		}
		s.client.log.Info().
			Str("raw", name).
			Str("canonical", resp.CanonicalName).
			Str("confidence", resp.Confidence).
			Msg("platform resolved via API")
		return resp.NetBoxID, nil
	}

	// Standalone mode
	return s.client.GetOrCreatePlatform(name, slug)
}

// mode returns a string describing current operating mode.
func (s *Syncer) mode() string {
	if s.isAPIMode() {
		return "api"
	}
	return "standalone"
}

// buildTags returns NetBox tags based on identifier source.
// Tags communicate exactly how this device was identified:
//
//	identified-by-motherboard-serial → no system serial, used motherboard serial
//	identified-by-mac                → no serial at all, used MAC address
//	identified-by-machine-id         → no serial or MAC, used machine-id (urgent)
func buildTags(identifierSource string) []map[string]string {
	var tags []map[string]string

	tags = append(tags, map[string]string{"slug": "syswatch"})

	switch identifierSource {
	case "mac-address":
		tags = append(tags, map[string]string{"slug": "identified-by-mac"})
	case "motherboard-serial":
		tags = append(tags, map[string]string{"slug": "identified-by-motherboard-serial"})
	case "machine-id":
		tags = append(tags, map[string]string{"slug": "identified-by-machine-id"})
	}

	return tags
}

// buildCustomFields maps hardware data to NetBox custom fields.
func buildCustomFields(hw models.Hardware) map[string]interface{} {
	fields := map[string]interface{}{}

	if hw.CPU.Model != "" {
		fields["cpu_model"] = hw.CPU.Model
	}
	if hw.CPU.TotalCores > 0 {
		fields["cpu_cores"] = hw.CPU.TotalCores
	}
	if len(hw.Memory) > 0 {
		fields["ram_gb"] = totalRAMFromModules(hw.Memory)
	}
	if hw.System.BIOSVersion != "" {
		fields["bios_version"] = hw.System.BIOSVersion
	}
	if hw.OS.Kernel != "" {
		fields["kernel"] = hw.OS.Kernel
	}
	fields["identifier_source"] = hw.Identifier.Source

	return fields
}

// buildInventoryPayloads creates inventory item payloads.
func buildInventoryPayloads(deviceID int, hw models.Hardware) []inventoryItemPayload {
	var items []inventoryItemPayload

	for i := 0; i < hw.CPU.Sockets; i++ {
		items = append(items, inventoryItemPayload{
			Device:      deviceID,
			Name:        fmt.Sprintf("CPU %d", i),
			Description: fmt.Sprintf("%s (%d cores)", hw.CPU.Model, hw.CPU.Cores),
			Discovered:  true,
		})
	}

	for _, dimm := range hw.Memory {
		items = append(items, inventoryItemPayload{
			Device:      deviceID,
			Name:        fmt.Sprintf("DIMM %s", dimm.Locator),
			Serial:      dimm.Serial,
			PartID:      dimm.PartNumber,
			Description: fmt.Sprintf("%dGB %s %dMHz", dimm.SizeGB, dimm.Type, dimm.SpeedMHz),
			Discovered:  true,
		})
	}

	for _, disk := range hw.Disks {
		items = append(items, inventoryItemPayload{
			Device:      deviceID,
			Name:        fmt.Sprintf("Disk %s", disk.Name),
			Serial:      disk.Serial,
			PartID:      disk.Model,
			Description: fmt.Sprintf("%dGB %s", disk.SizeGB, disk.Type),
			Discovered:  true,
		})
	}

	// Power Supplies
	for _, psu := range hw.PowerSupplies {
		desc := psu.Status
		if psu.MaxWatts > 0 {
			desc = fmt.Sprintf("%dW — %s", psu.MaxWatts, psu.Status)
		}
		items = append(items, inventoryItemPayload{
			Device:      deviceID,
			Name:        psu.Name,
			Serial:      psu.Serial,
			PartID:      psu.PartNumber,
			Description: desc,
			Discovered:  true,
		})
	}

	// GPUs
	for i, gpu := range hw.GPUs {
		name := fmt.Sprintf("GPU %d", i)
		desc := gpu.Address
		if gpu.Manufacturer != "" || gpu.Name != "" {
			desc = fmt.Sprintf("%s %s — %s", gpu.Manufacturer, gpu.Name, gpu.Address)
		}
		items = append(items, inventoryItemPayload{
			Device:      deviceID,
			Name:        name,
			Description: strings.TrimSpace(desc),
			Discovered:  true,
		})
	}

	return items
}

// buildNICItems converts models.NIC to nicItem.
func buildNICItems(nics []models.NIC) []nicItem {
	var items []nicItem
	for _, nic := range nics {
		items = append(items, nicItem{
			Name:       nic.Name,
			MACAddress: nic.MACAddress,
			Type:       nic.Type,
			IPAddress:  nic.IPAddress,
		})
	}
	return items
}

// totalRAMFromModules sums RAM across all DIMMs.
func totalRAMFromModules(modules []models.MemoryModule) int {
	total := 0
	for _, m := range modules {
		total += m.SizeGB
	}
	return total
}

// generateSlug converts a name to a NetBox compatible slug.
func generateSlug(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	slug := result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}
