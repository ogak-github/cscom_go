package metrics

import "github.com/shirou/gopsutil/v4/mem"

type RAM struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	UsedPct   float64 `json:"used_pct"`

	SwapTotal uint64  `json:"swap_total"`
	SwapUsed  uint64  `json:"swap_used"`
	SwapPct   float64 `json:"swap_pct"`
}

func collectRAM() (RAM, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return RAM{}, err
	}

	s, err := mem.SwapMemory()
	if err != nil {
		return RAM{}, err
	}

	return RAM{
		Total:     v.Total,
		Used:      v.Used,
		Available: v.Available,
		UsedPct:   v.UsedPercent,
		SwapTotal: s.Total,
		SwapUsed:  s.Used,
		SwapPct:   s.UsedPercent,
	}, nil
}
