package precheck

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ProbeScript runs on a node (via chroot /host) and prints KEY=VALUE lines.
//
//go:embed probe.sh
var ProbeScript string

// NodeHardware is the parsed output of ProbeScript.
type NodeHardware struct {
	Product        string
	Virt           string // systemd-detect-virt output; "none" on bare metal
	KVM            bool
	Threads        int
	MemTotalKB     int64
	MemAvailableKB int64
	Disks          []Disk
}

// BareMetal reports whether the node is not itself a VM.
func (h NodeHardware) BareMetal() bool {
	return h.Virt == "none"
}

// Disk is a whole block device on the node.
type Disk struct {
	Name       string
	SizeBytes  int64
	FSType     string
	PTType     string
	Mountpoint string
	Removable  bool
	Partitions int
	ByID       []string // /dev/disk/by-id names pointing at this disk
}

// Empty reports whether the disk has no partitions, filesystem, partition table or mount.
func (d Disk) Empty() bool {
	return d.Partitions == 0 && d.FSType == "" && d.PTType == "" && d.Mountpoint == "" && !d.Removable
}

// LVMMember reports whether the disk is an LVM physical volume (e.g. already taken by LVMS).
func (d Disk) LVMMember() bool {
	return d.FSType == "LVM2_member"
}

// SizeGB is the size in decimal gigabytes.
func (d Disk) SizeGB() int64 {
	return d.SizeBytes / 1_000_000_000
}

// PreferredByID returns the most stable /dev/disk/by-id path for the disk:
// wwn-* first, then scsi-3*/nvme-eui.*, then anything except LVM/DM aliases.
func (d Disk) PreferredByID() string {
	rank := func(id string) int {
		switch {
		case strings.HasPrefix(id, "wwn-"):
			return 0
		case strings.HasPrefix(id, "nvme-eui."), strings.HasPrefix(id, "scsi-3"):
			return 1
		case strings.HasPrefix(id, "lvm-pv-uuid-"), strings.HasPrefix(id, "dm-"):
			return 9
		default:
			return 2
		}
	}
	best, bestRank := "", 99
	ids := append([]string(nil), d.ByID...)
	sort.Strings(ids)
	for _, id := range ids {
		if r := rank(id); r < bestRank {
			best, bestRank = id, r
		}
	}
	if best == "" || bestRank == 9 {
		return ""
	}
	return "/dev/disk/by-id/" + best
}

// lsblk -J field types vary across util-linux versions (bool vs "0"/"1", number vs string).
type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	*b = flexBool(s == "true" || s == "1")
	return nil
}

type flexInt int64

func (n *flexInt) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		*n = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing size %q: %w", s, err)
	}
	*n = flexInt(v)
	return nil
}

type lsblkDevice struct {
	Name       string        `json:"name"`
	Size       flexInt       `json:"size"`
	Type       string        `json:"type"`
	FSType     *string       `json:"fstype"`
	PTType     *string       `json:"pttype"`
	Mountpoint *string       `json:"mountpoint"`
	RM         flexBool      `json:"rm"`
	Children   []lsblkDevice `json:"children"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ParseProbe parses ProbeScript output.
func ParseProbe(out string) (NodeHardware, error) {
	var hw NodeHardware
	byID := map[string][]string{}
	var lsblkJSON string

	for _, line := range strings.Split(out, "\n") {
		key, val, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		if key == "LSBLK" {
			lsblkJSON = val
			continue
		}
		applyProbeField(&hw, byID, key, val)
	}

	if hw.Virt == "" || hw.Threads == 0 {
		return hw, fmt.Errorf("probe output is missing VIRT/NPROC; got %d bytes", len(out))
	}
	if lsblkJSON == "" {
		return hw, nil
	}
	disks, err := parseLsblk(lsblkJSON, byID)
	if err != nil {
		return hw, err
	}
	hw.Disks = disks
	return hw, nil
}

func applyProbeField(hw *NodeHardware, byID map[string][]string, key, val string) {
	switch key {
	case "PRODUCT":
		hw.Product = val
	case "VIRT":
		hw.Virt = val
	case "KVM":
		hw.KVM = val == "yes"
	case "NPROC":
		hw.Threads, _ = strconv.Atoi(val)
	case "MEM_TOTAL_KB":
		hw.MemTotalKB, _ = strconv.ParseInt(val, 10, 64)
	case "MEM_AVAILABLE_KB":
		hw.MemAvailableKB, _ = strconv.ParseInt(val, 10, 64)
	case "BYID":
		if dev, id, ok := strings.Cut(val, " "); ok {
			byID[dev] = append(byID[dev], id)
		}
	}
}

// parseLsblk turns `lsblk -J` output into whole disks (loop/rom devices are skipped).
func parseLsblk(js string, byID map[string][]string) ([]Disk, error) {
	var parsed struct {
		BlockDevices []lsblkDevice `json:"blockdevices"`
	}
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		return nil, fmt.Errorf("parsing lsblk output: %w", err)
	}
	var disks []Disk
	for _, dev := range parsed.BlockDevices {
		if dev.Type != "disk" {
			continue
		}
		disks = append(disks, Disk{
			Name:       dev.Name,
			SizeBytes:  int64(dev.Size),
			FSType:     deref(dev.FSType),
			PTType:     deref(dev.PTType),
			Mountpoint: deref(dev.Mountpoint),
			Removable:  bool(dev.RM),
			Partitions: len(dev.Children),
			ByID:       byID[dev.Name],
		})
	}
	return disks, nil
}
