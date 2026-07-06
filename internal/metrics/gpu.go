package metrics

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GPU holds a point-in-time reading for one graphics card. UtilPct, TempC,
// and FanPct use -1 when unavailable. MemUsed/MemTotal/MemPct are only
// meaningful when MemTotal > 0 - callers must check that before using them.
// FanPct is only populated for NVIDIA (via nvidia-smi) - laptop dGPUs
// commonly report it as unsupported since the fan is EC-controlled, not
// driver-controlled. A GPU with its own hwmon node (mainly desktop amdgpu
// cards) shows up in the separate system-wide Fan list instead.
type GPU struct {
	Vendor   string  `json:"vendor"`
	Name     string  `json:"name"`
	UtilPct  float64 `json:"util_pct"`
	MemUsed  uint64  `json:"mem_used"`
	MemTotal uint64  `json:"mem_total"`
	MemPct   float64 `json:"mem_pct"`
	TempC    float64 `json:"temp_c"`
	FanPct   float64 `json:"fan_pct"`
}

// collectGPU gathers GPUs from every supported vendor. A missing tool or
// absent GPU is not an error - it just yields fewer entries.
func collectGPU() ([]GPU, error) {
	var gpus []GPU

	nv, err := collectNvidiaGPUs()
	if err != nil {
		return nil, err
	}
	gpus = append(gpus, nv...)
	gpus = append(gpus, collectSysfsGPUs()...)

	return gpus, nil
}

func collectNvidiaGPUs() ([]GPU, error) {
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, path,
		"--query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu,fan.speed",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, err
	}

	var gpus []GPU
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) != 6 {
			continue
		}
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}

		memUsedMB := parseFloatOrUnknown(fields[2])
		memTotalMB := parseFloatOrUnknown(fields[3])
		g := GPU{
			Vendor:  "NVIDIA",
			Name:    fields[0],
			UtilPct: parseFloatOrUnknown(fields[1]),
			TempC:   parseFloatOrUnknown(fields[4]),
			FanPct:  parseFloatOrUnknown(fields[5]),
		}
		if memTotalMB > 0 && memUsedMB >= 0 {
			g.MemUsed = uint64(memUsedMB) * 1024 * 1024
			g.MemTotal = uint64(memTotalMB) * 1024 * 1024
			g.MemPct = memUsedMB / memTotalMB * 100
		}
		gpus = append(gpus, g)
	}

	return gpus, nil
}

// parseFloatOrUnknown parses a numeric field, returning -1 for
// unsupported/unavailable values such as nvidia-smi's "[N/A]".
func parseFloatOrUnknown(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	return v
}

// collectSysfsGPUs reads AMD (amdgpu) and Intel (i915) GPUs directly from
// the kernel's DRM sysfs tree. NVIDIA's proprietary driver doesn't populate
// this, so it's handled separately via nvidia-smi.
func collectSysfsGPUs() []GPU {
	matches, _ := filepath.Glob("/sys/class/drm/card[0-9]*/device")
	var gpus []GPU

	for _, dir := range matches {
		vendor := strings.TrimSpace(readSysfsFile(filepath.Join(dir, "vendor")))

		var vendorName string
		switch vendor {
		case "0x1002":
			vendorName = "AMD"
		case "0x8086":
			vendorName = "Intel"
		default:
			continue
		}

		g := GPU{
			Vendor:  vendorName,
			Name:    vendorName + " GPU",
			UtilPct: -1,
			TempC:   -1,
			FanPct:  -1,
		}

		if busy := readSysfsFile(filepath.Join(dir, "gpu_busy_percent")); busy != "" {
			g.UtilPct = parseFloatOrUnknown(strings.TrimSpace(busy))
		}

		used, uerr := strconv.ParseUint(strings.TrimSpace(readSysfsFile(filepath.Join(dir, "mem_info_vram_used"))), 10, 64)
		total, terr := strconv.ParseUint(strings.TrimSpace(readSysfsFile(filepath.Join(dir, "mem_info_vram_total"))), 10, 64)
		if uerr == nil && terr == nil && total > 0 {
			g.MemUsed = used
			g.MemTotal = total
			g.MemPct = float64(used) / float64(total) * 100
		}

		// Only report this GPU if we actually got a real number for it -
		// a name-only row with no util and no memory is just noise.
		if g.UtilPct < 0 && g.MemTotal == 0 {
			continue
		}
		gpus = append(gpus, g)
	}

	return gpus
}

func readSysfsFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
