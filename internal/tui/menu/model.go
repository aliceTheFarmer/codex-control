package menu

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewMode int

const (
	viewList viewMode = iota
	viewActions
)

// Entry represents a selectable row in the menu.
type Entry struct {
	Number      int
	Title       string
	Subtitle    string
	Description string
	Badges      []string
	Payload     any
}

// Action executes when a user chooses an entry inside the action menu.
type Action struct {
	Label string
	Exec  func(entry Entry) tea.Cmd
}

// Config tunes the shared menu UI.
type Config struct {
	Context          context.Context
	LoadTimeout      time.Duration
	ListTitle        string
	ListHelp         []string
	ActionsTitle     string
	ActionsHelp      []string
	PanelPlaceholder string
	Loader           func(context.Context) ([]Entry, error)
	Actions          []Action
	DisablePanel     bool
}

// Result summarizes the completed interaction.
type Result struct {
	SelectedEntry *Entry
	ActionPayload any
	Message       string
	Success       bool
}

// Start launches the Bubble Tea program and blocks until completion.
func Start(cfg Config) (Result, error) {
	if cfg.Context == nil {
		cfg.Context = context.Background()
	}
	if cfg.LoadTimeout <= 0 {
		cfg.LoadTimeout = 20 * time.Second
	}
	m := model{
		cfg:        cfg,
		view:       viewList,
		loading:    true,
		message:    "Loading entries...",
		panelText:  cfg.PanelPlaceholder,
		panelTitle: "Information",
	}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	finished, ok := final.(model)
	if !ok {
		return Result{}, fmt.Errorf("unexpected model type %T", final)
	}
	return finished.toResult(), nil
}

type model struct {
	cfg Config

	entries      []Entry
	view         viewMode
	width        int
	height       int
	listCursor   int
	listOffset   int
	actionCursor int

	message     string
	panelTitle  string
	panelText   string
	numberInput string
	loading     bool

	lastAction *actionState
}

type actionState struct {
	entry   Entry
	payload any
	message string
	success bool
}

type entriesLoadedMsg struct {
	entries []Entry
	err     error
}

type panelMsg struct {
	title   string
	content string
	payload any
	err     error
}

func (m model) Init() tea.Cmd {
	return m.loadEntriesCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case entriesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.message = fmt.Sprintf("Failed to load entries: %v", msg.err)
			return m, nil
		}
		m.entries = msg.entries
		for i := range m.entries {
			m.entries[i].Number = i + 1
		}
		if len(m.entries) == 0 {
			m.listCursor = 0
		} else if m.listCursor >= len(m.entries) {
			m.listCursor = len(m.entries) - 1
		}
		m.ensureListCursorVisible()
		m.message = fmt.Sprintf("Loaded %d entries", len(m.entries))
		return m, nil
	case panelMsg:
		m.panelTitle = msg.title
		if msg.content == "" {
			m.panelText = "No output"
		} else {
			m.panelText = strings.TrimSpace(msg.content)
		}
		if msg.err != nil {
			m.message = fmt.Sprintf("%s failed: %v", msg.title, msg.err)
			m.lastAction = &actionState{entry: m.currentEntryValue(), payload: msg.payload, message: m.message, success: false}
		} else {
			m.message = fmt.Sprintf("%s ready", msg.title)
			m.lastAction = &actionState{entry: m.currentEntryValue(), payload: msg.payload, message: m.message, success: true}
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.view == viewActions {
			m.view = viewList
			m.actionCursor = 0
			m.numberInput = ""
			return m, nil
		}
		m.numberInput = ""
		return m, nil
	case "up", "k":
		if m.view == viewList {
			m.clearNumberInput()
			m.listCursor = clampCursor(m.listCursor-1, len(m.entries))
			m.ensureListCursorVisible()
			return m, nil
		}
		if m.view == viewActions {
			m.actionCursor = clampCursor(m.actionCursor-1, len(m.cfg.Actions))
		}
		return m, nil
	case "down", "j":
		if m.view == viewList {
			m.clearNumberInput()
			m.listCursor = clampCursor(m.listCursor+1, len(m.entries))
			m.ensureListCursorVisible()
			return m, nil
		}
		if m.view == viewActions {
			m.actionCursor = clampCursor(m.actionCursor+1, len(m.cfg.Actions))
		}
		return m, nil
	case "enter":
		if m.view == viewList {
			if m.numberInput != "" {
				idx, err := strconv.Atoi(m.numberInput)
				m.clearNumberInput()
				if err != nil || idx < 1 || idx > len(m.entries) {
					m.message = "Invalid selection"
					return m, nil
				}
				m.listCursor = idx - 1
				m.ensureListCursorVisible()
			}
			if len(m.entries) == 0 {
				return m, nil
			}
			m.view = viewActions
			m.actionCursor = 0
			return m, nil
		}
		if m.view == viewActions {
			if len(m.cfg.Actions) == 0 {
				return m, nil
			}
			entry := m.currentEntryValue()
			if entry.Title == "" {
				return m, nil
			}
			action := m.cfg.Actions[m.actionCursor]
			return m, action.Exec(entry)
		}
		return m, nil
	case "r", "R":
		if m.view == viewList {
			m.loading = true
			m.message = "Refreshing entries..."
			return m, m.loadEntriesCmd()
		}
		return m, nil
	}
	key := msg.String()
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		if m.view == viewList {
			if m.numberInput == "" && key == "0" {
				return m, nil
			}
			if len(m.numberInput) >= 4 {
				return m, nil
			}
			m.numberInput += key
			m.message = fmt.Sprintf("Jump target: %s", m.numberInput)
		}
		return m, nil
	}
	return m, nil
}

func (m *model) clearNumberInput() {
	m.numberInput = ""
}

func (m *model) ensureListCursorVisible() {
	total := len(m.entries)
	if total == 0 {
		m.listCursor = 0
		m.listOffset = 0
		return
	}
	if m.listCursor < 0 {
		m.listCursor = 0
	}
	if m.listCursor >= total {
		m.listCursor = total - 1
	}
	visible := m.listViewportSize()
	if visible <= 0 {
		visible = total
	}
	if visible > total {
		visible = total
	}
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.listOffset > maxOffset {
		m.listOffset = maxOffset
	}
	if m.listCursor < m.listOffset {
		m.listOffset = m.listCursor
	}
	if m.listCursor >= m.listOffset+visible {
		m.listOffset = m.listCursor - visible + 1
	}
	if m.listOffset < 0 {
		m.listOffset = 0
	}
}

func (m model) listViewportSize() int {
	if m.height <= 0 {
		return 20
	}
	size := m.height - 10
	if size < 6 {
		size = 6
	}
	return size
}

func (m model) currentEntryValue() Entry {
	if len(m.entries) == 0 {
		return Entry{}
	}
	if m.listCursor < 0 || m.listCursor >= len(m.entries) {
		return Entry{}
	}
	return m.entries[m.listCursor]
}

func (m model) loadEntriesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.cfg.Context, m.cfg.LoadTimeout)
		defer cancel()
		entries, err := m.cfg.Loader(ctx)
		return entriesLoadedMsg{entries: entries, err: err}
	}
}

func (m model) View() string {
	switch m.view {
	case viewList:
		return m.renderList()
	case viewActions:
		return m.renderActions()
	default:
		return ""
	}
}

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#C0CAF5"))
	panelStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	messageStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	prefixIdle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	prefixActive    = lipgloss.NewStyle().Foreground(lipgloss.Color("#8EACE3"))
	numberStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#A5B4FC")).Bold(true)
	badgeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#93C5FD"))
	entryTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5E7EB"))
	detailStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
)

func (m model) renderList() string {
	var left strings.Builder
	title := m.cfg.ListTitle
	if title == "" {
		title = "Entries"
	}
	left.WriteString(titleStyle.Render(title) + "\n")
	if len(m.cfg.ListHelp) > 0 {
		for _, line := range m.cfg.ListHelp {
			left.WriteString(line + "\n")
		}
		left.WriteString("\n")
	}
	if m.loading {
		left.WriteString("Loading entries...\n\n")
	}
	if len(m.entries) == 0 {
		left.WriteString("No entries available.\n")
	}
	start := m.listOffset
	visible := m.listViewportSize()
	if visible <= 0 || visible > len(m.entries) {
		visible = len(m.entries)
	}
	if start < 0 {
		start = 0
	}
	if len(m.entries) > 0 && start >= len(m.entries) {
		start = len(m.entries) - visible
		if start < 0 {
			start = 0
		}
	}
	end := start + visible
	if end > len(m.entries) {
		end = len(m.entries)
	}
	for i := start; i < end; i++ {
		entry := m.entries[i]
		pointer := prefixIdle.Render(" • ")
		if i == m.listCursor {
			pointer = prefixActive.Render(" › ")
		}
		number := numberStyle.Render(fmt.Sprintf("%3d.", entry.Number))
		badges := formatBadges(entry.Badges)
		titleText := entry.Title
		if titleText != "" {
			titleText = entryTitleStyle.Render(titleText)
		}
		summary := entry.Description
		if summary == "" {
			summary = entry.Subtitle
		}
		if summary == "" {
			summary = "(no description)"
		}
		summary = detailStyle.Render(summary)
		parts := []string{pointer, number, titleText}
		if badges != "" {
			parts = append(parts, badges)
		}
		parts = append(parts, "—", summary)
		left.WriteString(strings.Join(parts, " ") + "\n")
		if entry.Subtitle != "" && entry.Subtitle != entry.Description {
			left.WriteString(fmt.Sprintf("      %s\n", detailStyle.Render(entry.Subtitle)))
		}
	}
	left.WriteString("\n")
	if len(m.entries) > visible && visible > 0 {
		left.WriteString(messageStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.entries))) + "\n")
	}
	if m.numberInput != "" {
		left.WriteString(messageStyle.Render(fmt.Sprintf("Pending selection: %s", m.numberInput)) + "\n")
	}
	left.WriteString(messageStyle.Render(fmt.Sprintf("Status: %s", m.message)))
	if m.cfg.DisablePanel {
		return left.String()
	}
	right := m.renderPanel()
	return lipgloss.JoinHorizontal(lipgloss.Top, left.String(), right)
}

func (m model) renderActions() string {
	entry := m.currentEntryValue()
	var left strings.Builder
	title := m.cfg.ActionsTitle
	if title == "" {
		title = "Actions"
	}
	left.WriteString(titleStyle.Render(title) + "\n")
	if entry.Title != "" {
		subtitle := entry.Subtitle
		if subtitle == "" {
			subtitle = entry.Description
		}
		left.WriteString(fmt.Sprintf("Target: %s — %s\n", entry.Title, subtitle))
	}
	for _, line := range m.cfg.ActionsHelp {
		left.WriteString(line + "\n")
	}
	left.WriteString("\n")
	for i, action := range m.cfg.Actions {
		prefix := prefixIdle.Render(" • ")
		if i == m.actionCursor {
			prefix = prefixActive.Render(" › ")
		}
		left.WriteString(fmt.Sprintf("%s %s\n", prefix, action.Label))
	}
	left.WriteString("\n")
	left.WriteString(messageStyle.Render(fmt.Sprintf("Status: %s", m.message)))
	if m.cfg.DisablePanel {
		return left.String()
	}
	right := m.renderPanel()
	return lipgloss.JoinHorizontal(lipgloss.Top, left.String(), right)
}

func (m model) renderPanel() string {
	if m.cfg.DisablePanel {
		return ""
	}
	title := m.panelTitle
	if title == "" {
		title = "Information"
	}
	content := m.panelText
	if content == "" {
		content = m.cfg.PanelPlaceholder
	}
	if content == "" {
		content = "Select an action to view logs."
	}
	return panelStyle.Render(fmt.Sprintf("%s\n\n%s", titleStyle.Render(title), content))
}

func clampCursor(value, size int) int {
	if size == 0 {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value >= size {
		return size - 1
	}
	return value
}

// PanelUpdate builds a panel message for tea commands.
func PanelUpdate(title, content string, payload any, err error) tea.Msg {
	return panelMsg{title: title, content: content, payload: payload, err: err}
}

func (m model) toResult() Result {
	if m.lastAction == nil {
		return Result{Success: false, Message: "no action executed"}
	}
	entry := m.lastAction.entry
	return Result{
		SelectedEntry: &entry,
		ActionPayload: m.lastAction.payload,
		Message:       m.lastAction.message,
		Success:       m.lastAction.success,
	}
}

func formatBadges(badges []string) string {
	if len(badges) == 0 {
		return ""
	}
	rendered := make([]string, len(badges))
	for i, badge := range badges {
		rendered[i] = badgeStyle.Render(fmt.Sprintf("[%s]", badge))
	}
	return strings.Join(rendered, " ")
}
