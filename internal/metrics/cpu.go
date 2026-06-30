package metrics

import (
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

	return CPU{
		Model:   model,
		Cores:   len(percs),
		PerCore: percs,
		Load1:   avg.Load1,
		Load5:   avg.Load5,
		Load15:  avg.Load15,
		Uptime:  uptime,
	}, nil
}
