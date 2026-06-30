package metrics

import (
	"sort"

	gproc "github.com/shirou/gopsutil/v3/process"
)

const maxProcesses = 10

type Process struct {
	PID    int32   `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	MemRSS uint64  `json:"mem_rss"`
}

// collectProcesses reads per-process stats. cache holds live *gproc.Process
// objects keyed by PID so that CPUPercent() can compute accurate deltas.
func collectProcesses(cache map[int32]*gproc.Process) ([]Process, map[int32]*gproc.Process, error) {
	pids, err := gproc.Pids()
	if err != nil {
		return nil, cache, err
	}

	newCache := make(map[int32]*gproc.Process, len(pids))
	var result []Process

	for _, pid := range pids {
		p, ok := cache[pid]
		if !ok {
			p, err = gproc.NewProcess(pid)
			if err != nil {
				continue
			}
		}
		newCache[pid] = p

		cpuPct, err := p.CPUPercent()
		if err != nil {
			continue
		}
		name, _ := p.Name()
		mem, _ := p.MemoryInfo()

		var rss uint64
		if mem != nil {
			rss = mem.RSS
		}

		result = append(result, Process{
			PID:    pid,
			Name:   name,
			CPU:    cpuPct,
			MemRSS: rss,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CPU > result[j].CPU
	})
	if len(result) > maxProcesses {
		result = result[:maxProcesses]
	}

	return result, newCache, nil
}
