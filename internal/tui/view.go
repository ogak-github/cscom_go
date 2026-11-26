package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"yga.my.id/cscom/internal/metrics"
)

const (
	narrowWidth    = 60
	mediumWidth    = 90
	barWidthNarrow = 15
	barWidthMedium = 25
	barWidthWide   = 39

	// dockerImageWidth is the column budget for the ID+NAME+IMAGE+CPU%+MEM+
	// NET↓+NET↑ layout in renderDocker - it needs ~101 columns, so it can't
	// share the mediumWidth threshold used for the narrower NET-only tier.
	dockerImageWidth = 105

	// boxBorderWidth is the rounded border lipgloss draws on top of
	// whatever width we give it (1 column each side).
	boxBorderWidth = 2
	// boxPadding is the horizontal padding inside the border (1 column
	// each side), already included in the width passed to boxStyle.
	boxPadding = 2
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	critStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
)

// contentWidth is how much horizontal space a section's text actually has
// once a rounded box's border and padding are subtracted from the terminal
// width - bar/column sizing must be based on this, not the raw width.
func (m Model) contentWidth() int {
	w := m.width - boxBorderWidth - boxPadding
	if w < 1 {
		w = 1
	}
	return w
}

// boxWidth is the width to hand lipgloss's Width(), which already accounts
// for padding internally - the rounded border adds boxBorderWidth on top of
// this to reach the full terminal width.
func (m Model) boxWidth() int {
	w := m.width - boxBorderWidth
	if w < 1 {
		w = 1
	}
	return w
}

// renderBox wraps a section's content in a rounded box, or returns "" if
// the section had nothing to show - keeping the existing "hidden when
// empty" convention for optional metrics.
func renderBox(content string, width int) string {
	if content == "" {
		return ""
	}
	return boxStyle.Width(width).Render(strings.TrimRight(content, "\n"))
}

func (m Model) barWidth() int {
	w := m.contentWidth()
	switch {
	case w < narrowWidth:
		return barWidthNarrow
	case w < mediumWidth:
		return barWidthMedium
	default:
		return barWidthWide
	}
}

func (m Model) processNameWidth() int {
	w := m.contentWidth()
	switch {
	case w < narrowWidth:
		return 12
	case w < mediumWidth:
		return 16
	default:
		return 24
	}
}

func (m Model) compact() bool {
	return m.contentWidth() < narrowWidth
}

func (m Model) View() string {
	if !m.ready {
		return "  Loading..."
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf(" Control System Commander (CSCom) %s", m.version)) + "\n")
	b.WriteString(m.viewport.View() + "\n")
	b.WriteString(dimStyle.Render("  q quit | ↑↓ scroll | pgup/pgdn half page"))

	return b.String()
}

func fmtUptime(s uint64) string {
	d := s / 86400
	h := (s % 86400) / 3600
	m := (s % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func renderSystem(s metrics.System, c metrics.CPU, width int) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  System") + "\n")

	row := func(key, val string) {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			dimStyle.Render(fmt.Sprintf("%-10s", key)), val))
	}

	row("Platform", s.Platform)
	if s.Arch != "" {
		row("Arch", s.Arch)
	}
	row("Hostname", s.Hostname)
	if c.Uptime > 0 {
		row("Uptime", fmtUptime(c.Uptime))
	}
	if c.Model != "" {
		modelMax := 50
		if width < mediumWidth {
			modelMax = 30
		}
		row("CPU", truncate(c.Model, modelMax))
	}
	row("Cores", fmt.Sprintf("%d", c.Cores))
	return b.String()
}

func renderCPU(c metrics.CPU, bw int, compact bool) string {
	var b strings.Builder

	model := ""
	if c.Model != "" {
		modelMax := 40
		if compact {
			modelMax = 0
		} else if bw < barWidthMedium {
			modelMax = 20
		}
		if modelMax > 0 {
			model = "  " + dimStyle.Render(truncate(c.Model, modelMax))
		}
	}
	load := dimStyle.Render(fmt.Sprintf("  load: %.2f %.2f %.2f", c.Load1, c.Load5, c.Load15))
	temp := ""
	if c.TempC >= 0 {
		temp = dimStyle.Render(fmt.Sprintf("  temp: %.0f°C", c.TempC))
	}
	b.WriteString(headerStyle.Render("  CPU") + model + load + temp + "\n")

	for i, pct := range c.PerCore {
		if compact && i >= 8 {
			b.WriteString(fmt.Sprintf("  %s\n", dimStyle.Render(fmt.Sprintf("  ... %d more cores", len(c.PerCore)-8))))
			break
		}
		b.WriteString(fmt.Sprintf("  Core %-2d %s %s\n",
			i, coloredBar(pct, bw), pctColored(pct, fmt.Sprintf("%5.1f%%", pct))))
	}

	return b.String()
}

func renderRAM(r metrics.RAM, bw int) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("  RAM") + "\n")
	memInfo := dimStyle.Render(formatBytes(r.Used) + "/" + formatBytes(r.Total) + "  avail: " + formatBytes(r.Available))
	b.WriteString(fmt.Sprintf("  %-6s %s %s  %s\n",
		"Mem", coloredBar(r.UsedPct, bw),
		pctColored(r.UsedPct, fmt.Sprintf("%5.1f%%", r.UsedPct)),
		memInfo))

	if r.SwapTotal > 0 {
		swapFree := r.SwapTotal - r.SwapUsed
		swapInfo := dimStyle.Render(formatBytes(r.SwapUsed) + "/" + formatBytes(r.SwapTotal) + "  avail: " + formatBytes(swapFree))
		b.WriteString(fmt.Sprintf("  %-6s %s %s  %s\n",
			"Swap", coloredBar(r.SwapPct, bw),
			pctColored(r.SwapPct, fmt.Sprintf("%5.1f%%", r.SwapPct)),
			swapInfo))
	}

	return b.String()
}

func renderGPU(gpus []metrics.GPU, bw int) string {
	if len(gpus) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("  GPU") + "\n")

	for _, g := range gpus {
		b.WriteString(fmt.Sprintf("  %s\n", dimStyle.Render(g.Vendor+" "+g.Name)))

		if g.UtilPct >= 0 {
			b.WriteString(fmt.Sprintf("  %-6s %s %s\n",
				"Util", coloredBar(g.UtilPct, bw),
				pctColored(g.UtilPct, fmt.Sprintf("%5.1f%%", g.UtilPct))))
		} else {
			b.WriteString(fmt.Sprintf("  %-6s %s\n", "Util", dimStyle.Render("n/a")))
		}

		if g.MemTotal > 0 {
			memInfo := dimStyle.Render(formatBytes(g.MemUsed) + "/" + formatBytes(g.MemTotal))
			b.WriteString(fmt.Sprintf("  %-6s %s %s  %s\n",
				"Mem", coloredBar(g.MemPct, bw),
				pctColored(g.MemPct, fmt.Sprintf("%5.1f%%", g.MemPct)),
				memInfo))
		} else {
			b.WriteString(fmt.Sprintf("  %-6s %s\n", "Mem", dimStyle.Render("n/a")))
		}

		if g.TempC >= 0 {
			b.WriteString(fmt.Sprintf("  %-6s %s\n", "Temp", dimStyle.Render(fmt.Sprintf("%.0f°C", g.TempC))))
		}

		if g.FanPct >= 0 {
			b.WriteString(fmt.Sprintf("  %-6s %s\n", "Fan", dimStyle.Render(fmt.Sprintf("%.0f%%", g.FanPct))))
		}
	}

	return b.String()
}

func renderFans(fans []metrics.Fan) string {
	if len(fans) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("  Fans") + "\n")

	for _, f := range fans {
		b.WriteString(fmt.Sprintf("  %-20s %s\n",
			truncate(f.Label, 20), dimStyle.Render(fmt.Sprintf("%d RPM", f.RPM))))
	}

	return b.String()
}

func renderDisks(disks []metrics.Disk, bw int) string {
	if len(disks) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("  Disk") + "\n")

	for _, d := range disks {
		b.WriteString(fmt.Sprintf("  %-14s %s %s  %s\n",
			truncate(d.Mountpoint, 14),
			coloredBar(d.UsedPct, bw),
			pctColored(d.UsedPct, fmt.Sprintf("%5.1f%%", d.UsedPct)),
			dimStyle.Render(formatBytes(d.Used)+"/"+formatBytes(d.Total))))
	}

	return b.String()
}

func coloredBar(pct float64, bw int) string {
	blocks := bw
	if blocks < 1 {
		blocks = 1
	}

	clamped := pct
	if clamped < 0 {
		clamped = 0
	} else if clamped > 100 {
		clamped = 100
	}

	filled := int(math.Round(clamped / 100 * float64(blocks)))
	if clamped > 0 && filled == 0 {
		filled = 1
	}
	if filled > blocks {
		filled = blocks
	}

	warnAt := int(math.Ceil(0.60 * float64(blocks)))
	critAt := int(math.Ceil(0.80 * float64(blocks)))

	parts := make([]string, blocks)
	for i := 0; i < blocks; i++ {
		switch {
		case i >= filled:
			parts[i] = " "
		case i >= critAt:
			parts[i] = critStyle.Render("▮")
		case i >= warnAt:
			parts[i] = warnStyle.Render("▮")
		default:
			parts[i] = okStyle.Render("▮")
		}
	}

	return "[" + strings.Join(parts, "") + "]"
}

func pctColored(pct float64, s string) string {
	switch {
	case pct >= 80:
		return critStyle.Render(s)
	case pct >= 60:
		return warnStyle.Render(s)
	default:
		return okStyle.Render(s)
	}
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func renderNetwork(nets []metrics.Network, width int) string {
	if len(nets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Network") + "\n")

	showRate := width >= mediumWidth
	for _, n := range nets {
		if showRate {
			rx := okStyle.Render(fmt.Sprintf("↓ %8s/s", formatBytes(uint64(n.RXRate))))
			tx := dimStyle.Render(fmt.Sprintf("↑ %8s/s", formatBytes(uint64(n.TXRate))))
			b.WriteString(fmt.Sprintf("  %-12s  %s  %s\n", n.Name, rx, tx))
		} else {
			rx := okStyle.Render(fmt.Sprintf("↓ %s/s", formatBytes(uint64(n.RXRate))))
			tx := dimStyle.Render(fmt.Sprintf(" ↑ %s/s", formatBytes(uint64(n.TXRate))))
			b.WriteString(fmt.Sprintf("  %-12s %s%s\n", n.Name, rx, tx))
		}
	}
	return b.String()
}

func renderProcesses(procs []metrics.Process, nameWidth int) string {
	if len(procs) == 0 {
		return ""
	}

	// PSS is Linux-only and needs permission on the target process, so the
	// column is dropped entirely when nothing could be read - no point
	// showing a stripe of zeroes on macOS or an unprivileged run.
	showPSS := false
	for _, p := range procs {
		if p.MemPSS > 0 {
			showPSS = true
			break
		}
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("  Processes") + "\n")
	if showPSS {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %-7s  %-*s  %6s  %8s  %8s",
			"PID", nameWidth, "NAME", "CPU%", "RSS", "PSS")) + "\n")
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %-7s  %-*s  %6s  %8s",
			"PID", nameWidth, "NAME", "CPU%", "RSS")) + "\n")
	}

	for _, p := range procs {
		row := fmt.Sprintf("  %-7d  %-*s  %s  %8s",
			p.PID,
			nameWidth, truncate(p.Name, nameWidth),
			pctColored(p.CPU, fmt.Sprintf("%5.1f%%", p.CPU)),
			formatBytes(p.MemRSS))
		if showPSS {
			pss := dimStyle.Render(fmt.Sprintf("%8s", "-"))
			if p.MemPSS > 0 {
				pss = fmt.Sprintf("%8s", formatBytes(p.MemPSS))
			}
			row += "  " + pss
		}
		b.WriteString(row + "\n")
	}
	return b.String()
}

func renderDocker(containers []metrics.DockerContainer, bw int, width int) string {
	if len(containers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Docker") + "\n")

	showNet := width >= mediumWidth
	showImage := width >= dockerImageWidth

	nameW := 16
	imgW := 20
	if width < narrowWidth {
		nameW = 12
		imgW = 0
	}

	if showImage {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %-12s  %-*s  %-*s  %6s  %15s  %9s  %9s",
			"ID", nameW, "NAME", imgW, "IMAGE", "CPU%", "MEM", "NET↓", "NET↑")) + "\n")
	} else if showNet {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %-12s  %-*s  %6s  %15s  %9s  %9s",
			"ID", nameW, "NAME", "CPU%", "MEM", "NET↓", "NET↑")) + "\n")
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %-12s  %-*s  %6s  %9s",
			"ID", nameW, "NAME", "CPU%", "MEM")) + "\n")
	}

	for _, c := range containers {
		mem := formatBytes(c.MemUsed)
		if c.MemLimit > 0 {
			mem = formatBytes(c.MemUsed) + "/" + formatBytes(c.MemLimit)
		}
		if showImage {
			b.WriteString(fmt.Sprintf("  %-12s  %-*s  %-*s  %s  %15s  %9s  %9s\n",
				c.ID,
				nameW, truncate(c.Name, nameW),
				imgW, truncate(c.Image, imgW),
				pctColored(c.CPU, fmt.Sprintf("%5.1f%%", c.CPU)),
				mem,
				formatBytes(c.NetRX),
				formatBytes(c.NetTX)))
		} else if showNet {
			b.WriteString(fmt.Sprintf("  %-12s  %-*s  %s  %15s  %9s  %9s\n",
				c.ID,
				nameW, truncate(c.Name, nameW),
				pctColored(c.CPU, fmt.Sprintf("%5.1f%%", c.CPU)),
				mem,
				formatBytes(c.NetRX),
				formatBytes(c.NetTX)))
		} else {
			b.WriteString(fmt.Sprintf("  %-12s  %-*s  %s  %9s\n",
				c.ID,
				nameW, truncate(c.Name, nameW),
				pctColored(c.CPU, fmt.Sprintf("%5.1f%%", c.CPU)),
				mem))
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
