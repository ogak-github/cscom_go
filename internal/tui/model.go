package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"yga.my.id/cscom/internal/metrics"
)

type tickMsg time.Time

type Model struct {
	collector *metrics.Collector
	snap      metrics.Snapshot
	width     int
	height    int
	viewport  viewport.Model
	ready     bool
	version   string
}

func New(c *metrics.Collector, version string) Model {
	return Model{collector: c, snap: c.Latest(), version: version}
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.viewport.LineUp(1)
		case "down", "j":
			m.viewport.LineDown(1)
		case "pgup":
			m.viewport.HalfViewUp()
		case "pgdown":
			m.viewport.HalfViewDown()
		case "home":
			m.viewport.GotoTop()
		case "end":
			m.viewport.GotoBottom()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 2
		footerHeight := 1
		verticalMargin := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.SetContent(renderContent(m))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}
	case tickMsg:
		m.snap = m.collector.Latest()
		m.viewport.SetContent(renderContent(m))
		return m, tick()
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

const (
	// gridColMinWidth is the narrowest a grid column may get: enough for a
	// medium bar plus the RAM used/total/avail text on one line.
	gridColMinWidth = 80
	gridMaxCols     = 2
	gridGap         = 1
)

// renderContent stacks every section on narrow terminals. On wide ones the
// sections fill a grid row by row, left to right, and boxes sharing a row
// are stretched to the same height so the rows line up. Empty sections are
// skipped, so later ones move up to fill their slot.
func renderContent(m Model) string {
	cols := min(max(m.width/gridColMinWidth, 1), gridMaxCols)
	colW := (m.width - (cols-1)*gridGap) / cols
	cw := contentWidth(colW)
	bw := barWidth(cw)

	var sections []string
	for _, s := range []string{
		renderSystem(m.snap.System, m.snap.CPU, cw),
		renderCPU(m.snap.CPU, bw, cw < narrowWidth),
		renderRAM(m.snap.RAM, bw),
		renderGPU(m.snap.GPUs, bw),
		renderDisks(m.snap.Disks, bw),
		renderNetwork(m.snap.Networks, cw),
		renderFans(m.snap.Fans),
		renderProcesses(m.snap.Processes, processNameWidth(cw)),
		renderDocker(m.snap.Containers, bw, cw),
	} {
		if s != "" {
			sections = append(sections, s)
		}
	}

	var b strings.Builder
	for i := 0; i < len(sections); i += cols {
		row := sections[i:min(i+cols, len(sections))]

		// measure the rendered boxes (long lines wrap inside them), then
		// stretch every box in the row to the tallest one
		h := 0
		for _, s := range row {
			h = max(h, lipgloss.Height(renderBox(s, boxWidth(colW), 0)))
		}

		parts := make([]string, 0, 2*len(row))
		for j, s := range row {
			if j > 0 {
				parts = append(parts, strings.Repeat(" ", gridGap))
			}
			parts = append(parts, renderBox(s, boxWidth(colW), h-2)) // minus top+bottom border
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, parts...))
		b.WriteString("\n")
	}

	return b.String()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
