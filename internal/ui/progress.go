package ui

import (
	"fmt"
	"io"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fx64b/brenner/internal/device"
	"github.com/fx64b/brenner/internal/flash"
)

// WriteOp performs a write operation, calling report as it advances.
type WriteOp func(report func(flash.Progress)) error

type progressMsg flash.Progress

type doneMsg struct{ err error }

type progressModel struct {
	title string
	total uint64
	width int
	cur   flash.Progress
	start time.Time
	speed float64
	err   error
}

func newProgressModel(title string, total uint64) progressModel {
	return progressModel{title: title, total: total, width: 48, start: time.Now()}
}

func (m progressModel) Init() tea.Cmd { return nil }

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(10, min(msg.Width-6, 72))
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = ErrAborted
			return m, tea.Quit
		}
	case progressMsg:
		m.cur = flash.Progress(msg)
		if elapsed := time.Since(m.start).Seconds(); elapsed > 0 {
			m.speed = float64(m.cur.Written) / elapsed
		}
		return m, nil
	case doneMsg:
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m progressModel) View() string {
	frac := m.cur.Fraction()
	stats := fmt.Sprintf("%3.0f%%   %s/s   %s",
		frac*100, device.HumanSize(uint64(m.speed)), m.eta())
	sizes := fmt.Sprintf("%s / %s", device.HumanSize(m.cur.Written), device.HumanSize(m.total))
	return lipgloss.JoinVertical(lipgloss.Left,
		TitleStyle.Render(m.title),
		EmberBar(m.width, frac),
		SubtitleStyle.Render(stats+"   "+sizes),
	) + "\n"
}

func (m progressModel) eta() string {
	if m.speed <= 0 || m.cur.Written >= m.total {
		return "ETA --"
	}
	remaining := float64(m.total-m.cur.Written) / m.speed
	d := time.Duration(remaining * float64(time.Second)).Round(time.Second)
	return "ETA " + d.String()
}

// RunWithProgress renders an animated progress bar while op runs. Use it only
// when stdout is an interactive terminal.
func RunWithProgress(title string, total uint64, op WriteOp) error {
	p := tea.NewProgram(newProgressModel(title, total))
	go func() {
		err := op(func(pr flash.Progress) { p.Send(progressMsg(pr)) })
		p.Send(doneMsg{err: err})
	}()
	final, err := p.Run()
	if err != nil {
		return err
	}
	if fm, ok := final.(progressModel); ok {
		return fm.err
	}
	return nil
}

// PlainRun runs op without a TUI, printing a percentage line roughly every 5%.
// Used when stdout is not a terminal (scripts, pipes, CI).
func PlainRun(out io.Writer, title string, total uint64, op WriteOp) error {
	fmt.Fprintln(out, title)
	lastBucket := -1
	return op(func(pr flash.Progress) {
		pct := int(pr.Fraction() * 100)
		if bucket := pct / 5; bucket != lastBucket {
			lastBucket = bucket
			fmt.Fprintf(out, "  %3d%%   %s / %s\n", pct, device.HumanSize(pr.Written), device.HumanSize(total))
		}
	})
}
