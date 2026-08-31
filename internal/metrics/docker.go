package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DockerContainer holds per-container CPU%, memory, and network I/O.
type DockerContainer struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Image    string  `json:"image"`
	Status   string  `json:"status"`
	CPU      float64 `json:"cpu_pct"`
	MemUsed  uint64  `json:"mem_used"`
	MemLimit uint64  `json:"mem_limit"`
	MemPct   float64 `json:"mem_pct"`
	NetRX    uint64  `json:"net_rx"`
	NetTX    uint64  `json:"net_tx"`
}

var dockerClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", "/var/run/docker.sock")
		},
	},
	Timeout: 5 * time.Second,
}

type dockerListItem struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	Status string   `json:"Status"`
}

func collectDocker() ([]DockerContainer, error) {
	resp, err := dockerClient.Get("http://docker/containers/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list []dockerListItem
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}

	result := make([]DockerContainer, len(list))
	var wg sync.WaitGroup
	for i, item := range list {
		wg.Add(1)
		go func(idx int, c dockerListItem) {
			defer wg.Done()
			name := c.ID
			if len(c.ID) >= 12 {
				name = c.ID[:12]
			}
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			shortID := c.ID
			if len(c.ID) >= 12 {
				shortID = c.ID[:12]
			}
			dc := DockerContainer{
				ID:     shortID,
				Name:   name,
				Image:  c.Image,
				Status: c.Status,
			}
			if s, err := fetchContainerStats(c.ID); err == nil {
				dc.CPU = s.cpu
				dc.MemUsed = s.memUsed
				dc.MemLimit = s.memLimit
				if s.memLimit > 0 {
					dc.MemPct = float64(s.memUsed) / float64(s.memLimit) * 100
				}
				dc.NetRX = s.netRX
				dc.NetTX = s.netTX
			}
			result[idx] = dc
		}(i, item)
	}
	wg.Wait()

	return result, nil
}

type containerStats struct {
	cpu      float64
	memUsed  uint64
	memLimit uint64
	netRX    uint64
	netTX    uint64
}

func fetchContainerStats(id string) (containerStats, error) {
	resp, err := dockerClient.Get(fmt.Sprintf("http://docker/containers/%s/stats?stream=false&one-shot=true", id))
	if err != nil {
		return containerStats{}, err
	}
	defer resp.Body.Close()

	var s struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage  uint64   `json:"total_usage"`
				PercpuUsage []uint64 `json:"percpu_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs     int    `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
			Stats struct {
				Cache        uint64 `json:"cache"`
				InactiveFile uint64 `json:"inactive_file"`
			} `json:"stats"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return containerStats{}, err
	}

	var cs containerStats

	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemCPUUsage) - float64(s.PreCPUStats.SystemCPUUsage)
	numCPUs := s.CPUStats.OnlineCPUs
	if numCPUs == 0 {
		numCPUs = len(s.CPUStats.CPUUsage.PercpuUsage)
	}
	if numCPUs == 0 {
		numCPUs = 1
	}
	if sysDelta > 0 && cpuDelta >= 0 {
		cs.cpu = (cpuDelta / sysDelta) * float64(numCPUs) * 100.0
	}

	// prefer inactive_file (cgroup v2) over cache (cgroup v1) for RSS-like mem
	inactive := s.MemoryStats.Stats.InactiveFile
	if inactive == 0 {
		inactive = s.MemoryStats.Stats.Cache
	}
	cs.memUsed = s.MemoryStats.Usage
	if cs.memUsed > inactive {
		cs.memUsed -= inactive
	}
	cs.memLimit = s.MemoryStats.Limit

	for _, n := range s.Networks {
		cs.netRX += n.RxBytes
		cs.netTX += n.TxBytes
	}

	return cs, nil
}
