package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"yga.my.id/cscom/internal/metrics"
)

const (
	narrowWidth  = 60
	mediumWidth  = 90
	barWidthNarrow = 14
	barWidthMedium = 20
	barWidthWide   = 28
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	critStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func (m Model) barWidth() int {
	switch {
	case m.width < narrowWidth:
		return barWidthNarrow
	case m.width < mediumWidth:
		return barWidthMedium
	default:
		return barWidthWide
	}
}

func (m Model) processNameWidth() int {
	switch {
	case m.width < narrowWidth:
		return 12
	case m.width < mediumWidth:
		return 16
	default:
		return 24
	}
}

func (m Model) compact() bool {
	return m.width < narrowWidth
}

func (m Model) View() string {
	if !m.ready {
		return "  Loading..."
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Control System Commander (CSCom)") + "\n")
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
	b.WriteString(headerStyle.Render("  CPU") + model + load + "\n")

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
	filled := int(pct / 100 * float64(bw))
	if filled > bw {
		filled = bw
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", bw-filled)
	return "[" + pctColored(pct, bar) + "]"
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
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Processes") + "\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %-7s  %-*s  %6s  %8s",
		"PID", nameWidth, "NAME", "CPU%", "MEM")) + "\n")
	for _, p := range procs {
		b.WriteString(fmt.Sprintf("  %-7d  %-*s  %s  %8s\n",
			p.PID,
			nameWidth, truncate(p.Name, nameWidth),
			pctColored(p.CPU, fmt.Sprintf("%5.1f%%", p.CPU)),
			formatBytes(p.MemRSS)))
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
	showImage := width >= mediumWidth

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
