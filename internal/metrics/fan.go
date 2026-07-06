package metrics

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Fan is one hwmon-reported fan (CPU, case, or a GPU that registers its own
// hwmon node - e.g. desktop amdgpu cards). NVIDIA's GPU fan is reported as a
// percentage via nvidia-smi instead (see GPU.FanPct), since it doesn't use
// hwmon at all.
type Fan struct {
	Label string `json:"label"`
	RPM   int    `json:"rpm"`
}

// collectFans scans every hwmon chip for fan*_input files. Motherboards
// commonly expose several fan headers with nothing plugged into most of
// them, which read a permanent 0 RPM - those are dropped as noise unless
// the channel has a custom _label (a labeled channel is almost always
// actually wired to something, even if it's idle at 0 RPM right now).
func collectFans() []Fan {
	chips, _ := filepath.Glob("/sys/class/hwmon/hwmon*")
	var fans []Fan

	for _, dir := range chips {
		chipName := strings.TrimSpace(readSysfsFile(filepath.Join(dir, "name")))
		if chipName == "" {
			chipName = "fan"
		}

		inputs, _ := filepath.Glob(filepath.Join(dir, "fan*_input"))
		for _, f := range inputs {
			rpm, err := strconv.Atoi(strings.TrimSpace(readSysfsFile(f)))
			if err != nil {
				continue
			}

			base := strings.TrimSuffix(filepath.Base(f), "_input") // e.g. "fan1"
			customLabel := strings.TrimSpace(readSysfsFile(filepath.Join(dir, base+"_label")))
			if rpm == 0 && customLabel == "" {
				continue
			}

			label := chipName + " " + base
			if customLabel != "" {
				label = chipName + " " + customLabel
			}

			fans = append(fans, Fan{Label: label, RPM: rpm})
		}
	}

	return fans
}
