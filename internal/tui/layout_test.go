package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"yga.my.id/cscom/internal/metrics"
)

func layoutSnap() metrics.Snapshot {
	return metrics.Snapshot{
		System:    metrics.System{Hostname: "box", Platform: "linux", Arch: "x64"},
		CPU:       metrics.CPU{Model: "11th Gen Intel(R) Core(TM) i5-11320H @ 3.20GHz", Cores: 8, PerCore: []float64{10, 20, 30, 40, 50, 60, 70, 80}, TempC: 50},
		RAM:       metrics.RAM{Total: 16 << 30, Used: 8 << 30, Available: 8 << 30, UsedPct: 50},
		Disks:     []metrics.Disk{{Mountpoint: "/", Total: 100 << 30, Used: 50 << 30, UsedPct: 50}},
		Networks:  []metrics.Network{{Name: "eth0"}},
		Processes: []metrics.Process{{PID: 1, Name: "init", CPU: 1, MemRSS: 1 << 20}},
	}
}

// boxesOnFirstLine counts the rounded top-left corners on the first line,
// i.e. how many sections sit side by side at the top.
func boxesOnFirstLine(s string) int {
	return strings.Count(strings.SplitN(s, "\n", 2)[0], "╭")
}

func TestRenderContentLayout(t *testing.T) {
	tests := []struct {
		width, wantCols int
	}{
		{100, 1}, // narrow: stacked list
		{159, 1},
		{160, 2},
		{170, 2},
		{240, 2},
		{400, 2}, // capped
	}
	for _, tc := range tests {
		out := renderContent(Model{snap: layoutSnap(), width: tc.width})
		if got := boxesOnFirstLine(out); got != tc.wantCols {
			t.Errorf("width %d: %d boxes on first line, want %d", tc.width, got, tc.wantCols)
		}
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > tc.width {
				t.Fatalf("width %d: line %d is %d wide, overflows terminal", tc.width, i, w)
			}
		}
		if !strings.Contains(out, "Processes") {
			t.Errorf("width %d: processes section missing", tc.width)
		}

		// boxes sharing a row are padded to the same height, so every
		// bottom border line closes exactly wantCols boxes
		// (layoutSnap has 6 sections, an exact multiple of 1 and 2)
		for _, line := range strings.Split(out, "\n") {
			if n := strings.Count(line, "╰"); n != 0 && n != tc.wantCols {
				t.Errorf("width %d: row closes %d boxes, want %d: %q", tc.width, n, tc.wantCols, line)
				break
			}
		}
	}
}

func TestRenderContentRowOrder(t *testing.T) {
	out := renderContent(Model{snap: layoutSnap(), width: 200})
	first := out[:strings.Index(out, "╰")] // everything above the first row's bottom border
	if !strings.Contains(first, "System") || !strings.Contains(first, "CPU") {
		t.Errorf("first row should be System | CPU, got %q", first)
	}
	if strings.Index(out, "RAM") > strings.Index(out, "Disk") {
		t.Errorf("RAM should come before Disk")
	}
}

func TestRenderBoxCentersVertically(t *testing.T) {
	lines := strings.Split(renderBox("x", 10, 3), "\n")
	// top border, blank, content, blank, bottom border
	if len(lines) != 5 || !strings.Contains(lines[2], "x") {
		t.Errorf("content not vertically centered: %q", lines)
	}
}
