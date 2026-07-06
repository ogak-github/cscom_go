package metrics

import (
	"context"
	"sync"
	"time"

	gproc "github.com/shirou/gopsutil/v3/process"
)

// Snapshot is a point-in-time reading of all metrics.
type Snapshot struct {
	System     System            `json:"system"`
	CPU        CPU               `json:"cpu"`
	RAM        RAM               `json:"ram"`
	Disks      []Disk            `json:"disks"`
	Networks   []Network         `json:"networks"`
	Processes  []Process         `json:"processes"`
	Containers []DockerContainer `json:"containers"`
	GPUs       []GPU             `json:"gpus"`
	Fans       []Fan             `json:"fans"`
}

// Collector keeps a cached Snapshot refreshed in the background.
type Collector struct {
	mu        sync.RWMutex
	snap      Snapshot
	prevNet   *netSample
	procCache map[int32]*gproc.Process
}

// NewCollector creates a Collector and warms up the CPU percentage sampler.
// The first cpu.Percent call always returns 0; this discards it.
func NewCollector() *Collector {
	collectCPU()
	return &Collector{
		procCache: make(map[int32]*gproc.Process),
	}
}

// Refresh does an immediate synchronous collect and updates the cache.
func (c *Collector) Refresh() {
	c.refresh()
}

// Start begins background collection at the given interval until ctx is cancelled.
// Docker containers are collected on a separate 5-second cadence because
// the stats API call blocks ~1s per container.
func (c *Collector) Start(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.refresh()
			}
		}
	}()

	go func() {
		c.dockerRefresh()
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.dockerRefresh()
			}
		}
	}()

	go func() {
		c.gpuRefresh()
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.gpuRefresh()
			}
		}
	}()
}

// Latest returns a copy of the most recent Snapshot.
func (c *Collector) Latest() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snap
}

func (c *Collector) refresh() {
	sys, err := collectSystem()
	if err != nil {
		return
	}
	cpu, err := collectCPU()
	if err != nil {
		return
	}
	ram, err := collectRAM()
	if err != nil {
		return
	}
	disks, err := collectDisks()
	if err != nil {
		return
	}
	nets, newNetSample, _ := collectNetwork(c.prevNet)
	c.prevNet = newNetSample

	procs, newCache, _ := collectProcesses(c.procCache)
	c.procCache = newCache

	fans := collectFans()

	c.mu.Lock()
	containers := c.snap.Containers // preserve last Docker snapshot
	gpus := c.snap.GPUs             // preserve last GPU snapshot
	c.snap = Snapshot{
		System:     sys,
		CPU:        cpu,
		RAM:        ram,
		Disks:      disks,
		Networks:   nets,
		Processes:  procs,
		Containers: containers,
		GPUs:       gpus,
		Fans:       fans,
	}
	c.mu.Unlock()
}

func (c *Collector) dockerRefresh() {
	containers, err := collectDocker()
	if err != nil {
		return
	}
	c.mu.Lock()
	c.snap.Containers = containers
	c.mu.Unlock()
}

func (c *Collector) gpuRefresh() {
	gpus, err := collectGPU()
	if err != nil {
		return
	}
	c.mu.Lock()
	c.snap.GPUs = gpus
	c.mu.Unlock()
}
