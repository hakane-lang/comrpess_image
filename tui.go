package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// sessionState tracks which screen of the loop is currently active.
type sessionState int

const (
	stateForm sessionState = iota
	stateWorking
	stateResult
)

const (
	fieldInput = iota
	fieldOutput
	fieldQuality
	fieldCount
)

var (
	colorPrimary = lipgloss.Color("#7D56F4")
	colorAccent  = lipgloss.Color("#F25D94")
	colorSuccess = lipgloss.Color("#04B575")
	colorError   = lipgloss.Color("#FF5C5C")
	colorMuted   = lipgloss.Color("#6C6C6C")
	colorText    = lipgloss.Color("#DCDCDC")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Width(16)

	fieldBoxFocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	fieldBoxBlurred = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	successCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSuccess).
			Padding(1, 3)

	errorCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(1, 3)

	rowLabel = lipgloss.NewStyle().Foreground(colorMuted).Width(10)
	rowValue = lipgloss.NewStyle().Foreground(colorText)
)

// conversionResult carries the outcome of a compress operation between goroutines.
type conversionResult struct {
	inPath, outPath string
	inSize, outSize int64
	err             error
}

type resultMsg conversionResult

type model struct {
	state      sessionState
	inputs     []textinput.Model
	focusIndex int
	spinner    spinner.Model
	formErr    string
	result     *conversionResult
	quitting   bool
}

func newModel() model {
	in := textinput.New()
	in.Placeholder = "photo.jpg"
	in.Prompt = ""
	in.Focus()

	out := textinput.New()
	out.Placeholder = "photo.webp"
	out.Prompt = ""

	qual := textinput.New()
	qual.Placeholder = "90"
	qual.Prompt = ""
	qual.CharLimit = 3

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return model{
		state:   stateForm,
		inputs:  []textinput.Model{in, out, qual},
		spinner: s,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

		switch m.state {
		case stateForm:
			return m.updateForm(msg)
		case stateResult:
			switch msg.String() {
			case "q", "esc":
				m.quitting = true
				return m, tea.Quit
			case "enter", "n":
				return m.resetForm(), nil
			}
		}

	case spinner.TickMsg:
		if m.state == stateWorking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case resultMsg:
		res := conversionResult(msg)
		m.result = &res
		m.state = stateResult
		return m, nil
	}

	return m, nil
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.quitting = true
		return m, tea.Quit

	case "tab", "down":
		m.focusIndex = (m.focusIndex + 1) % fieldCount
		m.focusInputs()
		return m, nil

	case "shift+tab", "up":
		m.focusIndex = (m.focusIndex - 1 + fieldCount) % fieldCount
		m.focusInputs()
		return m, nil

	case "enter":
		return m.submit()
	}

	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	return m, cmd
}

func (m *model) focusInputs() {
	for i := range m.inputs {
		if i == m.focusIndex {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m model) submit() (tea.Model, tea.Cmd) {
	in := strings.TrimSpace(m.inputs[fieldInput].Value())
	out := strings.TrimSpace(m.inputs[fieldOutput].Value())
	qualStr := strings.TrimSpace(m.inputs[fieldQuality].Value())
	if qualStr == "" {
		qualStr = "90"
	}

	if in == "" {
		m.formErr = "input path is required"
		return m, nil
	}
	if out == "" {
		m.formErr = "output path is required"
		return m, nil
	}

	quality, err := strconv.Atoi(qualStr)
	if err != nil || quality < 0 || quality > 100 {
		m.formErr = "quality must be a number between 0 and 100"
		return m, nil
	}

	m.formErr = ""
	m.state = stateWorking
	return m, tea.Batch(m.spinner.Tick, convertCmd(in, out, quality))
}

func (m model) resetForm() model {
	nm := newModel()
	// Keep the previous output directory hint by reusing quality only.
	nm.inputs[fieldQuality].SetValue(m.inputs[fieldQuality].Value())
	return nm
}

func convertCmd(in, out string, quality int) tea.Cmd {
	return func() tea.Msg {
		res := conversionResult{inPath: in, outPath: out}

		if size, err := fileSize(in); err == nil {
			res.inSize = size
		}

		img, err := LoadImage(in)
		if err != nil {
			res.err = err
			return resultMsg(res)
		}

		if err := SaveImage(img, out, quality); err != nil {
			res.err = err
			return resultMsg(res)
		}

		if size, err := fileSize(out); err == nil {
			res.outSize = size
		}

		return resultMsg(res)
	}
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	header := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("🗜  COMPRESS"),
		subtitleStyle.Render("Image compression & conversion, in your terminal."),
	)

	var body string
	switch m.state {
	case stateForm:
		body = m.viewForm()
	case stateWorking:
		body = m.viewWorking()
	case stateResult:
		body = m.viewResult()
	}

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, "", body))
}

func (m model) viewForm() string {
	labels := []string{"Input file", "Output file", "Quality (0-100)"}

	rows := make([]string, 0, len(m.inputs))
	for i, ti := range m.inputs {
		box := fieldBoxBlurred
		if i == m.focusIndex {
			box = fieldBoxFocused
		}
		row := lipgloss.JoinHorizontal(lipgloss.Center,
			labelStyle.Render(labels[i]),
			box.Render(ti.View()),
		)
		rows = append(rows, row)
	}

	form := lipgloss.JoinVertical(lipgloss.Left, rows...)

	var errLine string
	if m.formErr != "" {
		errLine = "\n" + errorStyle.Render("✗ "+m.formErr)
	}

	help := helpStyle.Render("tab/shift+tab move • enter convert • esc quit")

	return form + errLine + "\n" + help
}

func (m model) viewWorking() string {
	return fmt.Sprintf("%s Compressing your image...", m.spinner.View())
}

func (m model) viewResult() string {
	res := m.result
	if res.err != nil {
		content := lipgloss.JoinVertical(lipgloss.Left,
			errorStyle.Render("✗ Conversion failed"),
			"",
			rowValue.Render(res.err.Error()),
		)
		card := errorCard.Render(content)
		help := helpStyle.Render("enter try again • q quit")
		return card + "\n" + help
	}

	saved := float64(0)
	if res.inSize > 0 {
		saved = (1 - float64(res.outSize)/float64(res.inSize)) * 100
	}

	savedStyle := lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	if saved < 0 {
		savedStyle = lipgloss.NewStyle().Bold(true).Foreground(colorError)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(colorSuccess).Render("✓ Saved successfully"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, rowLabel.Render("Input"), rowValue.Render(fmt.Sprintf("%s (%s)", res.inPath, humanSize(res.inSize)))),
		lipgloss.JoinHorizontal(lipgloss.Left, rowLabel.Render("Output"), rowValue.Render(fmt.Sprintf("%s (%s)", res.outPath, humanSize(res.outSize)))),
		lipgloss.JoinHorizontal(lipgloss.Left, rowLabel.Render("Saved"), savedStyle.Render(fmt.Sprintf("%.1f%%", saved))),
	)

	card := successCard.Render(content)
	help := helpStyle.Render("enter compress another • q quit")
	return card + "\n" + help
}
