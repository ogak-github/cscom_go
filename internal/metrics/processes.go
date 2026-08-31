package metrics

import (
	"os"
	"sort"
	"strconv"
	"strings"

	gproc "github.com/shirou/gopsutil/v3/process"
)

const maxProcesses = 10

type Process struct {
	PID  int32   `json:"pid"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"`
	// MemRSS is the resident set size - every page the process has in RAM,
	// including pages shared with other processes (libc, mmap'd files). It
	// matches htop's RES, but summing it across processes double-counts
	// shared memory.
	MemRSS uint64 `json:"mem_rss"`
	// MemPSS is the proportional set size - private pages plus this
	// process's fair share of every shared page. It is the honest "how much
	// RAM does this app actually cost" number, and summing it across all
	// processes stays close to real usage. Linux-only, and 0 when the
	// kernel is too old for smaps_rollup or we lack permission to read
	// another user's process.
	MemPSS uint64 `json:"mem_pss"`
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

	// PSS is only filled in for the processes we actually display: reading
	// smaps_rollup makes the kernel walk the process's page tables, which is
	// far too expensive to do for every PID on the system each tick.
	for i := range result {
		result[i].MemPSS = readPSS(result[i].PID)
	}

	return result, newCache, nil
}

// readPSS pulls the Pss line out of /proc/<pid>/smaps_rollup (kernel 4.14+).
// Returns 0 on any failure - a missing file (non-Linux, older kernel, process
// already gone) or EACCES on a process owned by another user.
func readPSS(pid int32) uint64 {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(int(pid)) + "/smaps_rollup")
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(data), "\n") {
		// The line looks like "Pss:  12345 kB". Match the exact key so the
		// Pss_Anon / Pss_File / Pss_Shmem breakdown lines are skipped.
		key, val, ok := strings.Cut(line, ":")
		if !ok || key != "Pss" {
			continue
		}
		fields := strings.Fields(val)
		if len(fields) == 0 {
			continue
		}
		kb, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return 0
		}
		return kb * 1024
	}
	return 0
}
