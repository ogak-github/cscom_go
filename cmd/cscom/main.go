package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"yga.my.id/cscom/internal/metrics"
	"yga.my.id/cscom/internal/tui"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collector := metrics.NewCollector()
	collector.Refresh()
	collector.Start(ctx, time.Second)

	p := tea.NewProgram(tui.New(collector), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
