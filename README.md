# syswatch

Automated hardware inventory for NetBox. Runs on any Linux server, collects hardware via SMBIOS, and syncs to NetBox automatically.

Single binary. Zero dependencies. Works on bare metal and VMs.

---

## What It Collects

| Category | Details |
|----------|---------|
| System | Manufacturer, model, serial number |
| CPU | Model, cores, sockets |
| Memory | DIMM count, size, type, speed |
| Disks | Name, size, type, serial |
| Network | Interface name, MAC, IP |
| OS | Name, version, kernel |

---

## Modes

| Mode | Description |
|------|-------------|
| `api` | Uses [syswatch-api](https://github.com/avvvet/syswatch-api) for intelligent name resolution. **Default and recommended.** |
| `standalone` | Writes directly to NetBox. Limited matching. Not recommended for production. |

Set in `/etc/syswatch/.env`:
```bash
SYSWATCH_MODE=api         # default — requires syswatch-api running
SYSWATCH_MODE=standalone  # no syswatch-api needed, limited matching
```

---

## Quick Install

```bash
coming
```

---

## Manual Install

```bash
# Download binary
curl -fsSL https://github.com/avvvet/syswatch/releases/latest/download/syswatch-server \
  -o /usr/local/bin/syswatch
chmod +x /usr/local/bin/syswatch

# Configure
mkdir -p /etc/syswatch
cp .env.example /etc/syswatch/.env
vi /etc/syswatch/.env

# Run
sudo syswatch
```

> Root is required for SMBIOS hardware access.

---

## Configuration

```bash
# NetBox
NETBOX_URL=http://netbox.company.com
NETBOX_TOKEN=Bearer nbt_your_token
NETBOX_SITE=NYC-DC01
NETBOX_ROLE=server

# Mode
SYSWATCH_MODE=api   # api or standalone

# Central Intelligence API (when MODE=api)
SYSWATCH_API_URL=http://syswatch-api:8080
SYSWATCH_API_KEY=your-api-key
```

---

## Device Identification

syswatch uses a fallback chain to uniquely identify each server:

```
1. SMBIOS serial number       → smbios-serial
2. Motherboard serial number  → identified-by-motherboard-serial (tag)
3. MAC address                → identified-by-mac (tag)
4. /etc/machine-id            → identified-by-machine-id (tag)
```

Device name in NetBox: `hostname-serial` (e.g. `web01-5CD237789L`)

---

## Tags

| Tag | Priority | Meaning |
|-----|----------|---------|
| `syswatch` | — | Managed by syswatch |
| `identified-by-motherboard-serial` | Low | No system serial |
| `identified-by-mac` | Medium | No serial, used MAC |
| `identified-by-machine-id` | High | No serial or MAC — fix urgently |

---

## NetBox Requirements

**Site** must exist before running syswatch — never auto-created.

**Custom fields** on Device:

| Field | Type |
|-------|------|
| `cpu_model` | Text |
| `cpu_cores` | Integer |
| `ram_gb` | Integer |
| `bios_version` | Text |
| `kernel` | Text |
| `identifier_source` | Text |

---

## Schedule

```bash
# Run every 6 hours via cron
0 */6 * * * root /usr/local/bin/syswatch >> /var/log/syswatch.log 2>&1
```

---

## Build

```bash
git clone https://github.com/avvvet/syswatch
cd syswatch
go mod tidy
make build
```

---

## Related

- [syswatch-api](https://github.com/avvvet/syswatch-api) — Central Intelligence API
- [netbox-community/devicetype-library](https://github.com/netbox-community/devicetype-library) — community device type definitions