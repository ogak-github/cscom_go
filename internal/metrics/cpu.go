package metrics

import (
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
)

type CPU struct {
	Model   string    `json:"model"`
	Cores   int       `json:"cores"`
	PerCore []float64 `json:"per_core"`
	Load1   float64   `json:"load1"`
	Load5   float64   `json:"load5"`
	Load15  float64   `json:"load15"`
	Uptime  uint64    `json:"uptime"`
	TempC   float64   `json:"temp_c"` // -1 if unknown
}

func collectCPU() (CPU, error) {
	percs, err := cpu.Percent(0, true)
	if err != nil {
		return CPU{}, err
	}

	avg, err := load.Avg()
	if err != nil {
		return CPU{}, err
	}

	infos, _ := cpu.Info()
	model := ""
	if len(infos) > 0 {
		model = infos[0].ModelName
	}

	uptime, _ := host.Uptime()

	// SensorsTemperatures commonly returns a non-nil error alongside valid
	// partial results (e.g. one unreadable sensor among many) - the data
	// that did come back is still useful, so the error itself is ignored.
	temps, _ := host.SensorsTemperatures()

	return CPU{
		Model:   model,
		Cores:   len(percs),
		PerCore: percs,
		Load1:   avg.Load1,
		Load5:   avg.Load5,
		Load15:  avg.Load15,
		Uptime:  uptime,
		TempC:   selectCPUTemp(temps),
	}, nil
}

// selectCPUTemp picks the single most representative "CPU temperature" out
// of whatever hwmon sensors gopsutil found, since the key naming differs by
// vendor/platform. Returns -1 if nothing recognizable was found.
func selectCPUTemp(temps []host.TemperatureStat) float64 {
	var coreSum float64
	var coreCount int

	for _, t := range temps {
		switch {
		case t.SensorKey == "coretemp_package_id_0":
			return t.Temperature
		case strings.Contains(t.SensorKey, "k10temp") && strings.Contains(t.SensorKey, "tctl"):
			return t.Temperature
		case strings.HasPrefix(t.SensorKey, "coretemp_core_"):
			coreSum += t.Temperature
			coreCount++
		}
	}
	if coreCount > 0 {
		return coreSum / float64(coreCount)
	}

	for _, t := range temps {
		if strings.Contains(t.SensorKey, "k10temp") {
			return t.Temperature
		}
	}
	for _, t := range temps {
		switch t.SensorKey {
		case "cpu_thermal", "cpu-thermal", "soc_thermal":
			return t.Temperature
		}
	}

	return -1
}
