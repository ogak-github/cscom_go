package metrics

import (
	"testing"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/sensors"
)

func TestSkipPartition(t *testing.T) {
	tests := []struct {
		mount, fstype, device string
		skip                  bool
	}{
		{"/", "ext4", "/dev/nvme0n1p2", false},
		{"/boot/efi", "vfat", "/dev/nvme0n1p1", false},
		{"/snap/core/1", "squashfs", "/dev/loop0", true},
		{"/run", "tmpfs", "tmpfs", true},
		// macOS (taken from a real M1 run)
		{"/", "apfs", "/dev/disk3s1s1", false},
		{"/Volumes/External", "apfs", "/dev/disk5s1", false},
		{"/dev", "devfs", "devfs", true},
		{"/System/Volumes/Data", "apfs", "/dev/disk3s5", true},
		{"/System/Volumes/VM", "apfs", "/dev/disk3s6", true},
		{"/private/var/run/com.apple.security.cryptexd/mnt/x", "apfs", "/dev/disk7s1", true},
		{"/Users/me/Library/Developer/CoreDevice/DeviceFS", "devicefs", "devicefs", true},
		{"/Library/Developer/CoreSimulator/Volumes/iOS_22C150", "apfs", "/dev/disk9s1", true},
	}
	for _, tc := range tests {
		got := skipPartition(disk.PartitionStat{Mountpoint: tc.mount, Fstype: tc.fstype, Device: tc.device})
		if got != tc.skip {
			t.Errorf("skipPartition(%s %s) = %v, want %v", tc.mount, tc.fstype, got, tc.skip)
		}
	}
}

func TestSelectCPUTempAppleSilicon(t *testing.T) {
	temps := []sensors.TemperatureStat{
		{SensorKey: "gas gauge battery", Temperature: 37.3},
		{SensorKey: "GPU MTR Temp Sensor4", Temperature: 30},
		{SensorKey: "pACC MTR Temp Sensor5", Temperature: 64},
		{SensorKey: "pACC MTR Temp Sensor7", Temperature: 66},
		{SensorKey: "eACC MTR Temp Sensor0", Temperature: 56},
		{SensorKey: "PMU tdie7", Temperature: 62},
	}
	if got := selectCPUTemp(temps); got != 62 {
		t.Errorf("CPU cluster average = %v, want 62", got)
	}

	// chips without the ACC sensors still expose PMU die temps
	tdie := []sensors.TemperatureStat{
		{SensorKey: "PMU tdie1", Temperature: 50},
		{SensorKey: "PMU2 tdie8", Temperature: 60},
		{SensorKey: "PMU tdev2", Temperature: 40},
	}
	if got := selectCPUTemp(tdie); got != 55 {
		t.Errorf("tdie average = %v, want 55", got)
	}

	if got := selectCPUTemp([]sensors.TemperatureStat{{SensorKey: "gas gauge battery", Temperature: 37}}); got != -1 {
		t.Errorf("unknown sensors = %v, want -1", got)
	}
}
