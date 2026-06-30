package metrics

import (
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
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
}

func collectDisks() ([]Disk, error) {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var disks []Disk
	for _, p := range parts {
		if pseudoFS[p.Fstype] {
			continue
		}
		if strings.HasPrefix(p.Device, "/dev/loop") {
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
