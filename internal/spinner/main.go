package spinner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Context struct {
	context.Context
	buf bytes.Buffer
	p   *tea.Program
}

func (c *Context) Write(p []byte) (n int, err error) {
	for i, b := range p {
		if b == '\n' || b == '\r' {
			c.p.Send(output(c.buf.String()))
			c.buf.Reset()
			continue
		}

		if err := c.buf.WriteByte(b); err != nil {
			return i, err
		}
	}

	return len(p), nil
}

func (c *Context) SetTitle(s string) {
	c.p.Send(title(s))
}

func Run(ctx context.Context, title string, f func(ctx *Context) error) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	m := spinnerModel{
		spinner: s,
		title:   title,
	}

	progCtx, cancel := context.WithCancel(ctx)

	program := tea.NewProgram(m)
	ictx := &Context{Context: progCtx, p: program}

	var wg sync.WaitGroup
	wg.Go(func() {
		program.Run()
		cancel()
	})

	err := f(ictx)
	program.Send(msgComplete{})

	wg.Wait()
	return err
}

type (
	msgComplete struct{}
	title       string
	output      string
)

type spinnerModel struct {
	spinner spinner.Model
	title   string
	output  string
	done    bool
	err     error
}

// Init is the first function that will be called. It returns an optional
// initial command. To not perform an initial command return nil.
func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update is called when a message is received. Use it to inspect messages
// and, in response, update the model and/or send a command.
func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.err = errors.New("stopped by user")
			return m, tea.Quit
		}
	case msgComplete:
		m.done = true
		return m, tea.Quit
	case title:
		m.title = string(msg)
		return m, nil
	case output:
		m.output = string(msg)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

var gray = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

// View renders the program's UI, which can be a string or a [Layer]. The
// view is rendered after every Update.
func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	if m.output != "" {
		return fmt.Sprintf("%s%s\n   %s", m.spinner.View(), m.title, gray.Render(m.output))
	}

	return m.spinner.View() + m.title
}
