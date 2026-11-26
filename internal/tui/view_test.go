package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"yga.my.id/cscom/internal/metrics"
)

func TestColoredBarHtop(t *testing.T) {
	tests := []struct {
		name      string
		pct       float64
		bw        int
		wantWidth int
		wantFill  int
	}{
		{"zero narrow", 0, barWidthNarrow, 17, 0},
		{"low positive narrow", 2, barWidthNarrow, 17, 1},
		{"half medium", 50, barWidthMedium, 27, 13},
		{"warn medium", 75, barWidthMedium, 27, 19},
		{"full wide", 100, barWidthWide, 41, 39},
		{"negative clamped", -10, barWidthMedium, 27, 0},
		{"over 100 clamped", 130, barWidthMedium, 27, 25},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := coloredBar(tc.pct, tc.bw)
			if w := lipgloss.Width(got); w != tc.wantWidth {
				t.Fatalf("lipgloss.Width(%q) = %d, want %d", got, w, tc.wantWidth)
			}
			if !strings.HasPrefix(got, "[") || !strings.HasSuffix(got, "]") {
				t.Fatalf("bar missing brackets: %q", got)
			}
			if gotFill := strings.Count(got, "▮"); gotFill != tc.wantFill {
				t.Fatalf("filled blocks = %d, want %d (bar=%q)", gotFill, tc.wantFill, got)
			}
		})
	}
}

func TestRenderSectionsUseHtopIndicators(t *testing.T) {
	cpuOut := renderCPU(metrics.CPU{
		Model:   "Test CPU",
		PerCore: []float64{10, 85},
		TempC:   45,
	}, barWidthMedium, false)
	if !strings.Contains(cpuOut, "[") || !strings.Contains(cpuOut, "▮") {
		t.Errorf("renderCPU missing htop bar indicators: %s", cpuOut)
	}

	ramOut := renderRAM(metrics.RAM{
		Total:     16 * 1024 * 1024 * 1024,
		Used:      8 * 1024 * 1024 * 1024,
		Available: 8 * 1024 * 1024 * 1024,
		UsedPct:   50,
		SwapTotal: 4 * 1024 * 1024 * 1024,
		SwapUsed:  1 * 1024 * 1024 * 1024,
		SwapPct:   25,
	}, barWidthMedium)
	if !strings.Contains(ramOut, "[") || !strings.Contains(ramOut, "▮") {
		t.Errorf("renderRAM missing htop bar indicators: %s", ramOut)
	}

	gpuOut := renderGPU([]metrics.GPU{{
		Vendor:   "NVIDIA",
		Name:     "RTX",
		UtilPct:  65,
		MemTotal: 8 * 1024 * 1024 * 1024,
		MemUsed:  4 * 1024 * 1024 * 1024,
		MemPct:   50,
		TempC:    60,
		FanPct:   40,
	}}, barWidthMedium)
	if !strings.Contains(gpuOut, "[") || !strings.Contains(gpuOut, "▮") {
		t.Errorf("renderGPU missing htop bar indicators: %s", gpuOut)
	}

	diskOut := renderDisks([]metrics.Disk{{
		Mountpoint: "/",
		Used:       50 * 1024 * 1024 * 1024,
		Total:      100 * 1024 * 1024 * 1024,
		UsedPct:    50,
	}}, barWidthMedium)
	if !strings.Contains(diskOut, "[") || !strings.Contains(diskOut, "▮") {
		t.Errorf("renderDisks missing htop bar indicators: %s", diskOut)
	}
}
