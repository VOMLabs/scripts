package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"scripts/internal/executor"
	"scripts/internal/openrouter"
)

type outputMsg struct {
	name string
	line string
}

type scriptDone struct {
	name string
	err  error
}

type orResult string
type orError struct{ err error }

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#3B82F6")).
			Padding(0, 1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).
			Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).
			Bold(true)

	doneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	orOnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).
			Bold(true)

	orOffStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	idleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	labelStyles = []lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#F472B6")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FB923C")).Bold(true),
	}
)

type runInstance struct {
	cancel context.CancelFunc
	err    error
}

type Model struct {
	scripts     []executor.Script
	cursor      int
	listOffset  int
	showSidebar bool
	running     map[string]*runInstance
	outputLines []string
	vpOffset    int
	vpHeight    int
	orEnabled   bool
	apiKey      string
	width       int
	height      int
	ready       bool
	inputMode   bool
	inputArgs   string
	inputTarget string
}

func New(scripts []executor.Script, apiKey string) Model {
	return Model{
		scripts:     scripts,
		apiKey:      apiKey,
		showSidebar: true,
		running:     make(map[string]*runInstance),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.vpHeight = m.height - 5
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if m.inputMode {
			return m.handleInputKey(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			for _, inst := range m.running {
				if inst.cancel != nil {
					inst.cancel()
				}
			}
			return m, tea.Quit
		case "s":
			m.showSidebar = !m.showSidebar
			return m, nil
		}

		if !m.showSidebar {
			switch msg.String() {
			case "up", "k":
				if m.vpOffset > 0 {
					m.vpOffset--
				}
			case "down", "j":
				if m.vpOffset < len(m.outputLines)-m.vpHeight {
					m.vpOffset++
				}
			case "pgup":
				m.vpOffset -= m.vpHeight / 2
				if m.vpOffset < 0 {
					m.vpOffset = 0
				}
			case "pgdown":
				m.vpOffset += m.vpHeight / 2
				maxOffset := len(m.outputLines) - m.vpHeight
				if maxOffset < 0 {
					maxOffset = 0
				}
				if m.vpOffset > maxOffset {
					m.vpOffset = maxOffset
				}
			case "g":
				m.vpOffset = 0
			case "G":
				m.vpOffset = len(m.outputLines) - m.vpHeight
				if m.vpOffset < 0 {
					m.vpOffset = 0
				}
			case "r":
				m.orEnabled = !m.orEnabled
			}
			return m, nil
		}

		// Sidebar navigation + output scrolling
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.listOffset {
					m.listOffset = m.cursor
				}
			}
		case "down", "j":
			if m.cursor < len(m.scripts)-1 {
				m.cursor++
				maxVisible := m.height - 8
				if m.cursor-m.listOffset >= maxVisible {
					m.listOffset = m.cursor - maxVisible + 1
				}
			}
		case "enter":
			if len(m.scripts) > 0 {
				s := m.scripts[m.cursor]
				m.inputMode = true
				m.inputArgs = ""
				m.inputTarget = s.Name
				return m, nil
			}
		case "pgup":
			m.vpOffset -= m.vpHeight / 2
			if m.vpOffset < 0 {
				m.vpOffset = 0
			}
		case "pgdown":
			m.vpOffset += m.vpHeight / 2
			maxOffset := len(m.outputLines) - m.vpHeight
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.vpOffset > maxOffset {
				m.vpOffset = maxOffset
			}
		case "g":
			m.vpOffset = 0
		case "G":
			m.vpOffset = len(m.outputLines) - m.vpHeight
			if m.vpOffset < 0 {
				m.vpOffset = 0
			}
		case "r":
			m.orEnabled = !m.orEnabled
		}

	case outputMsg:
		line := fmt.Sprintf("[%s] %s", msg.name, msg.line)
		m.outputLines = append(m.outputLines, line)
		if m.vpOffset >= len(m.outputLines)-m.vpHeight-1 {
			m.vpOffset = len(m.outputLines) - m.vpHeight
			if m.vpOffset < 0 {
				m.vpOffset = 0
			}
		}
		return m, nil

	case scriptDone:
		if inst, ok := m.running[msg.name]; ok {
			inst.err = msg.err
		}
		delete(m.running, msg.name)
		exitLine := fmt.Sprintf("[%s] exited", msg.name)
		if msg.err != nil {
			exitLine = fmt.Sprintf("[%s] error: %s", msg.name, msg.err)
		}
		m.outputLines = append(m.outputLines, exitLine)
		m.vpOffset = len(m.outputLines) - m.vpHeight
		if m.vpOffset < 0 {
			m.vpOffset = 0
		}
		executor.Cleanup(msg.name)
		if m.orEnabled && m.apiKey != "" {
			return m, processWithOR(m.outputLines, m.apiKey)
		}
		return m, nil

	case orResult:
		lines := strings.Split(string(msg), "\n")
		m.outputLines = append(m.outputLines, "", "--- OpenRouter Output ---")
		for _, l := range lines {
			m.outputLines = append(m.outputLines, l)
		}
		m.vpOffset = len(m.outputLines) - m.vpHeight
		if m.vpOffset < 0 {
			m.vpOffset = 0
		}
		return m, nil

	case orError:
		m.outputLines = append(m.outputLines, "", fmt.Sprintf("--- OpenRouter Error: %s ---", msg.err))
		return m, nil
	}

	return m, nil
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		for _, inst := range m.running {
			if inst.cancel != nil {
				inst.cancel()
			}
		}
		return m, tea.Quit
	case "enter":
		if m.inputTarget == "" {
			m.inputMode = false
			return m, nil
		}
		var s *executor.Script
		for i := range m.scripts {
			if m.scripts[i].Name == m.inputTarget {
				s = &m.scripts[i]
				break
			}
		}
		if s == nil {
			m.inputMode = false
			return m, nil
		}
		if _, ok := m.running[s.Name]; ok {
			m.inputMode = false
			return m, nil
		}
		args := parseArgs(m.inputArgs)
		ctx, cancel := context.WithCancel(context.Background())
		m.running[s.Name] = &runInstance{cancel: cancel}
		startLine := fmt.Sprintf("[%s] started", s.Name)
		if len(args) > 0 {
			startLine = fmt.Sprintf("[%s] started with: %s", s.Name, m.inputArgs)
		}
		m.outputLines = append(m.outputLines, startLine)
		m.vpOffset = len(m.outputLines) - m.vpHeight
		if m.vpOffset < 0 {
			m.vpOffset = 0
		}
		m.inputMode = false
		return m, runScript(ctx, s.Name, *s, args)

	case "esc":
		m.inputMode = false
		return m, nil

	case "backspace":
		if len(m.inputArgs) > 0 {
			m.inputArgs = m.inputArgs[:len(m.inputArgs)-1]
		}
		return m, nil

	case "space":
		m.inputArgs += " "
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.inputArgs += msg.String()
		}
		return m, nil
	}
}

func parseArgs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.Fields(s)
}

func runScript(ctx context.Context, name string, s executor.Script, args []string) tea.Cmd {
	base := func(line string) outputMsg { return outputMsg{name: name, line: line} }
	done := func(err error) scriptDone { return scriptDone{name: name, err: err} }

	if s.Type == executor.TypeCompilable {
		return runCompilable(ctx, s, args, base, done)
	}
	return runDirect(ctx, s, args, base, done)
}

func runCompilable(ctx context.Context, s executor.Script, args []string, base func(string) outputMsg, done func(error) scriptDone) tea.Cmd {
	compileCmd := executor.CompileCommand(s)
	cmd := exec.CommandContext(ctx, compileCmd.Path, compileCmd.Args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return func() tea.Msg { return done(err) }
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return func() tea.Msg { return done(err) }
	}
	if err := cmd.Start(); err != nil {
		return func() tea.Msg { return done(err) }
	}

	reader := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(reader)
	compileDone := make(chan error, 1)
	go func() { compileDone <- cmd.Wait() }()

	type phase int
	const (
		phaseCompile phase = iota
		phaseRun
		phaseDone
	)

	p := phaseCompile
	var runScanner *bufio.Scanner
	var runDone chan error

	return func() tea.Msg {
		switch p {
		case phaseCompile:
			if scanner.Scan() {
				return base(scanner.Text())
			}
			if err := <-compileDone; err != nil {
				p = phaseDone
				return done(fmt.Errorf("compile failed: %w", err))
			}
			p = phaseRun
			return base("--- Compilation OK, running ---")

		case phaseRun:
			if runScanner == nil {
				runCmd := executor.RunCompiledCmd(s, args...)
				runCmd = exec.CommandContext(ctx, runCmd.Path, runCmd.Args[1:]...)
				rStdout, err := runCmd.StdoutPipe()
				if err != nil {
					p = phaseDone
					return done(err)
				}
				rStderr, err := runCmd.StderrPipe()
				if err != nil {
					p = phaseDone
					return done(err)
				}
				if err := runCmd.Start(); err != nil {
					p = phaseDone
					return done(err)
				}
				rReader := io.MultiReader(rStdout, rStderr)
				runScanner = bufio.NewScanner(rReader)
				runDone = make(chan error, 1)
				go func() { runDone <- runCmd.Wait() }()
			}
			if runScanner.Scan() {
				return base(runScanner.Text())
			}
			p = phaseDone
			err := <-runDone
			return done(err)

		case phaseDone:
			return nil
		}
		return nil
	}
}

func runDirect(ctx context.Context, s executor.Script, args []string, base func(string) outputMsg, done func(error) scriptDone) tea.Cmd {
	cmd, err := executor.Command(s, args...)
	if err != nil {
		return func() tea.Msg { return done(err) }
	}
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return func() tea.Msg { return done(err) }
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return func() tea.Msg { return done(err) }
	}
	if err := cmd.Start(); err != nil {
		return func() tea.Msg { return done(err) }
	}

	reader := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(reader)
	doneCh := make(chan error, 1)
	go func() { doneCh <- cmd.Wait() }()

	var finished bool
	return func() tea.Msg {
		if finished {
			return nil
		}
		if scanner.Scan() {
			return base(scanner.Text())
		}
		finished = true
		err := <-doneCh
		return done(err)
	}
}

func processWithOR(lines []string, apiKey string) tea.Cmd {
	return func() tea.Msg {
		text := strings.Join(lines, "\n")
		result, err := openrouter.SendText(text, apiKey)
		if err != nil {
			return orError{err: err}
		}
		return orResult(result)
	}
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var body string
	statusWidth := m.width

	if m.showSidebar {
		listWidth := m.width * 3 / 10
		outputWidth := m.width - listWidth
		if listWidth < 20 {
			listWidth = 20
		}
		if outputWidth < 30 {
			outputWidth = 30
		}
		listPanel := m.renderList(listWidth, m.height-4)
		outputPanel := m.renderOutput(outputWidth, m.height-4)
		body = lipgloss.JoinHorizontal(lipgloss.Top, listPanel, outputPanel)
	} else {
		outputPanel := m.renderOutput(m.width, m.height-4)
		body = outputPanel
	}

	statusBar := m.renderStatus(statusWidth)
	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
}

func (m Model) renderList(width, height int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Width(width - 2).Render(" Scripts "))
	b.WriteString("\n\n")

	maxVisible := height - 3
	if maxVisible < 1 {
		maxVisible = 1
	}

	start := m.listOffset
	end := start + maxVisible
	if end > len(m.scripts) {
		end = len(m.scripts)
	}

	displayed := m.scripts[start:end]
	for i, s := range displayed {
		label := s.Name
		if s.Type == executor.TypeCompilable {
			label += " [C]"
		}

		_, isRunning := m.running[s.Name]
		line := "  " + label
		if start+i == m.cursor {
			line = "▸ " + label
			if isRunning {
				line = cursorStyle.Render(line) + " " + runningStyle.Render("●")
			} else {
				line = cursorStyle.Render(line)
			}
		} else if isRunning {
			line = "  " + label + " " + runningStyle.Render("●")
		}

		b.WriteString(line)
		if i < len(displayed)-1 {
			b.WriteString("\n")
		}
	}

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B82F6"))
	return style.Render(b.String())
}

func (m Model) renderOutput(width, height int) string {
	var b strings.Builder

	runningCount := len(m.running)
	title := " Output "
	if runningCount > 0 {
		title = fmt.Sprintf(" Output (%d running) ", runningCount)
	}
	b.WriteString(titleStyle.Width(width - 2).Render(title))
	b.WriteString("\n\n")

	if len(m.outputLines) == 0 {
		b.WriteString("Select a script and press Enter to run it.")
	} else {
		start := m.vpOffset
		end := start + height - 3
		if end > len(m.outputLines) {
			end = len(m.outputLines)
		}
		if start < 0 {
			start = 0
		}
		// Assign a consistent color index per script name
		nameIndex := map[string]int{}
		idx := 0
		for i := start; i < end; i++ {
			line := m.outputLines[i]
			// Try to extract the [name] prefix
			if strings.HasPrefix(line, "[") {
				if closeIdx := strings.Index(line, "]"); closeIdx > 0 {
					label := line[1:closeIdx]
					if _, ok := nameIndex[label]; !ok {
						nameIndex[label] = idx
						idx++
					}
					style := labelStyles[nameIndex[label]%len(labelStyles)]
					line = style.Render("["+label+"]") + line[closeIdx+1:]
				}
			}
			b.WriteString(line)
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B82F6"))
	return style.Render(b.String())
}

func (m Model) renderStatus(width int) string {
	if m.inputMode {
		prompt := fmt.Sprintf(" Args for %s: %s█ ", m.inputTarget, m.inputArgs)
		return lipgloss.NewStyle().
			Width(width).
			Background(lipgloss.Color("#1E3A5F")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Render(prompt)
	}

	runningCount := len(m.running)
	runningNames := ""
	if runningCount > 0 {
		names := make([]string, 0, runningCount)
		for name := range m.running {
			names = append(names, name)
		}
		runningNames = strings.Join(names, ", ")
	}

	state := idleStyle.Render("Idle")
	if runningCount > 0 {
		state = runningStyle.Render(fmt.Sprintf("Running (%d)", runningCount))
	}

	orLabel := orOffStyle.Render("OR: OFF")
	if m.orEnabled {
		orLabel = orOnStyle.Render("OR: ON")
	}

	left := fmt.Sprintf(" %s | %s | %s ", runningNames, state, orLabel)
	right := " s sidebar | Enter run | r toggle OR | q quit "

	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 1 {
		padding = 1
	}
	sep := strings.Repeat(" ", padding)

	return lipgloss.NewStyle().
		Width(width).
		Background(lipgloss.Color("#1F2937")).
		Foreground(lipgloss.Color("#D1D5DB")).
		Render(left + sep + right)
}
