package metrics

import (
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

type Disk struct {
	Mountpoint string  `json:"mountpoint"`
	Fstype     string  `json:"fstype"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Free       uint64  `json:"free"`
	UsedPct    float64 `json:"used_pct"`
}

var pseudoFS = map[string]bool{
	"tmpfs": true, "devtmpfs": true, "sysfs": true, "proc": true,
	"devpts": true, "cgroup": true, "cgroup2": true, "pstore": true,
	"mqueue": true, "debugfs": true, "hugetlbfs": true, "securityfs": true,
	"configfs": true, "fusectl": true, "bpf": true, "tracefs": true,
	"overlay": true, "squashfs": true, "efivarfs": true, "autofs": true,
	"devfs": true, "devicefs": true, // macOS
}

// macOS mounts its sealed system volumes, cryptexes and simulator runtimes
// as separate APFS volumes; the user only cares about / and /Volumes/*.
var hiddenMountPrefixes = []string{
	"/System/Volumes/",
	"/private/var/",
	"/Library/Developer/CoreSimulator/",
}

func skipPartition(p disk.PartitionStat) bool {
	if pseudoFS[p.Fstype] || strings.HasPrefix(p.Device, "/dev/loop") {
		return true
	}
	for _, prefix := range hiddenMountPrefixes {
		if strings.HasPrefix(p.Mountpoint, prefix) {
			return true
		}
	}
	return false
}

func collectDisks() ([]Disk, error) {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var disks []Disk
	for _, p := range parts {
		if skipPartition(p) {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		disks = append(disks, Disk{
			Mountpoint: p.Mountpoint,
			Fstype:     p.Fstype,
			Total:      usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			UsedPct:    usage.UsedPercent,
		})
	}
	return disks, nil
}
