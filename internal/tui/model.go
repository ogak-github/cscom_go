package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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

func renderContent(m Model) string {
	cw := m.contentWidth()
	bw := m.boxWidth()

	sections := []string{
		renderSystem(m.snap.System, m.snap.CPU, cw),
		renderCPU(m.snap.CPU, m.barWidth(), m.compact()),
		renderFans(m.snap.Fans),
		renderRAM(m.snap.RAM, m.barWidth()),
		renderGPU(m.snap.GPUs, m.barWidth()),
		renderDisks(m.snap.Disks, m.barWidth()),
		renderNetwork(m.snap.Networks, cw),
		renderProcesses(m.snap.Processes, m.processNameWidth()),
		renderDocker(m.snap.Containers, m.barWidth(), cw),
	}

	var b strings.Builder
	for _, s := range sections {
		if boxed := renderBox(s, bw); boxed != "" {
			b.WriteString(boxed)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
