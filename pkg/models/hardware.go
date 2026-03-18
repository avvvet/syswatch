package models

// Hardware represents the complete hardware inventory of a server.
// This is the central data structure passed between the collector
// and the NetBox sync layer.
type Hardware struct {
	System     System
	Identifier Identifier
	CPU        CPU
	Memory     []MemoryModule
	Disks      []Disk
	NICs       []NIC
	OS         OS
}

// System holds top level server identity.
type System struct {
	Manufacturer      string // e.g. Dell
	Model             string // e.g. PowerEdge R750
	Serial            string // e.g. SRV123456
	MotherboardSerial string // fallback if Serial is empty
	UHeight           int    // rack units e.g. 1 or 2
	BIOSVersion       string // e.g. 2.15.0
}

// Identifier holds the resolved unique hardware identifier.
// Tracks which fallback source was used so NetBox can be
// tagged accordingly for human review.
type Identifier struct {
	Value  string // the actual identifier value
	Source string // smbios-serial, motherboard-serial, mac-address, machine-id
}

// CPU holds processor information.
type CPU struct {
	Model      string // e.g. Intel Xeon Gold 6338
	Cores      int    // physical cores per socket
	Sockets    int    // number of populated sockets
	TotalCores int    // Cores * Sockets
}

// MemoryModule represents a single DIMM slot.
type MemoryModule struct {
	Locator      string // slot label e.g. DIMM A1
	Manufacturer string // e.g. Samsung
	PartNumber   string // e.g. M393A4K40DB3
	Serial       string // per DIMM serial
	SizeGB       int    // size in GB
	Type         string // e.g. DDR4
	SpeedMHz     int    // e.g. 3200
}

// Disk represents a single storage device.
type Disk struct {
	Name         string // kernel name e.g. sda, nvme0n1
	Manufacturer string // e.g. Samsung
	Model        string // e.g. PM983
	Serial       string // per disk serial
	SizeGB       int    // size in GB
	Type         string // SSD, HDD, NVMe
}

// NIC represents a single network interface.
type NIC struct {
	Name       string // kernel name e.g. eth0
	MACAddress string // e.g. AA:BB:CC:DD:EE:FF
	SpeedMbps  int    // e.g. 25000 for 25GbE
	Type       string // NetBox type e.g. 25gbase-x-sfp28
	IPAddress  string // e.g. 10.0.1.25/24 if assigned
}

// OS holds operating system information.
type OS struct {
	Name    string // e.g. Ubuntu
	Version string // e.g. 22.04
	Kernel  string // e.g. 5.15.0-91-generic
	Slug    string // e.g. ubuntu-22-04
}
