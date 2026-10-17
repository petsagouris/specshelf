package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/petsagouris/specshelf/internal/vault"
)

// mode is which view the session is showing.
type mode int

const (
	modeList mode = iota
	modeDetail
	modeCreate
	modeHelp
)

// loadedMsg is the result of one List call: the documents, the warnings for the
// ones that could not be read, and any error from the store.
type loadedMsg struct {
	records  []vault.IndexRecord
	warnings []vault.Warning
	err      error
}

// detailMsg is one document's bytes, fetched for the detail view.
type detailMsg struct {
	ref     string
	content string
	err     error
}

// Model is the interactive session's state and its three views plus help.
type Model struct {
	store Store

	records  []vault.IndexRecord
	warnings []vault.Warning
	err      error

	width, height int
	mode          mode
	cursor        int

	// filter narrows the loaded documents in memory; filtering is true while
	// the text query is being typed.
	filter    Filter
	filtering bool

	// detailRef is the reference of the document open in the detail view and
	// detail its loaded content.
	detailRef string
	detail    string

	// createKind and createTitle are the create prompt's captured input.
	createKind  string
	createTitle string

	// Create is the request the create prompt submitted. It is kept so 5.4 can
	// route it through the store; the prompt itself does not mutate the vault.
	Create *vault.NewDocument

	quitting bool
}

// New returns a session model over store, at a conventional initial size until
// the program reports the terminal's real one.
func New(store Store) *Model {
	return &Model{store: store, width: 80, height: 24, createKind: "spec"}
}

// Init loads the document list.
func (m *Model) Init() tea.Cmd { return m.loadCmd() }

// loadCmd reads the vault through the store. It is the only operation that
// re-reads from disk; filtering happens in memory.
func (m *Model) loadCmd() tea.Cmd {
	store := m.store
	if store == nil {
		return nil
	}
	return func() tea.Msg {
		records, warnings, err := store.List()
		return loadedMsg{records: records, warnings: warnings, err: err}
	}
}

// Update handles the messages the session reacts to.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg:
		m.records, m.warnings, m.err = msg.records, msg.warnings, msg.err
		m.clampCursor()
		return m, nil
	case detailMsg:
		if msg.ref == m.detailRef {
			m.detail = msg.content
			if msg.err != nil {
				m.detail = "error: " + msg.err.Error()
			}
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

// View renders the active view. Every view is a full-screen one.
func (m *Model) View() tea.View {
	var content string
	switch m.mode {
	case modeDetail:
		content = m.renderDetail()
	case modeCreate:
		content = m.renderCreate()
	case modeHelp:
		content = m.renderHelp()
	default:
		content = m.renderList()
	}
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

// handleKey routes a key press to the active view. It returns a command when
// the key asks for work (a reload, a detail fetch, or quitting).
func (m *Model) handleKey(key tea.KeyPressMsg) tea.Cmd {
	switch m.mode {
	case modeDetail:
		return m.handleDetailKey(key)
	case modeCreate:
		return m.handleCreateKey(key)
	case modeHelp:
		return m.handleHelpKey(key)
	default:
		return m.handleListKey(key)
	}
}

func (m *Model) handleListKey(key tea.KeyPressMsg) tea.Cmd {
	if m.filtering {
		return m.handleFilterKey(key)
	}
	switch key.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return tea.Quit
	case "j", "down":
		m.move(1)
	case "k", "up":
		m.move(-1)
	case "enter":
		return m.openDetail()
	case "r":
		return m.loadCmd()
	case "c":
		m.mode = modeCreate
		m.createTitle = ""
	case "?":
		m.mode = modeHelp
	case "/":
		m.filtering = true
	case "K":
		m.filter.Kind = next(kindOptions(m.records), m.filter.Kind)
		m.clampCursor()
	case "S":
		m.filter.Status = next(statusOptions(m.records), m.filter.Status)
		m.clampCursor()
	case "R":
		m.filter.Repo = next(repoOptions(m.records), m.filter.Repo)
		m.clampCursor()
	case "T":
		m.filter.Tag = next(tagOptions(m.records), m.filter.Tag)
		m.clampCursor()
	}
	return nil
}

// handleFilterKey edits the text query in memory. Enter keeps the query and
// leaves editing; esc clears it; every printable key narrows the list.
func (m *Model) handleFilterKey(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "ctrl+c":
		m.quitting = true
		return tea.Quit
	case "enter":
		m.filtering = false
	case "esc":
		m.filtering = false
		m.filter.Query = ""
	case "backspace":
		m.filter.Query = dropLastRune(m.filter.Query)
	default:
		if key.Text != "" {
			m.filter.Query += key.Text
		}
	}
	m.clampCursor()
	return nil
}

func (m *Model) handleDetailKey(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return tea.Quit
	case "esc":
		m.mode = modeList
	}
	return nil
}

func (m *Model) handleHelpKey(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return tea.Quit
	case "esc", "?":
		m.mode = modeList
	}
	return nil
}

func (m *Model) handleCreateKey(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "ctrl+c":
		m.quitting = true
		return tea.Quit
	case "esc":
		m.mode = modeList
	case "tab":
		if m.createKind == "spec" {
			m.createKind = "decision"
		} else {
			m.createKind = "spec"
		}
	case "backspace":
		m.createTitle = dropLastRune(m.createTitle)
	case "enter":
		// 5.4 attaches here: the store gains a Create method and this request
		// is handed to it; until then the prompt only captures it.
		m.Create = &vault.NewDocument{Kind: m.createKind, Title: strings.TrimSpace(m.createTitle)}
		m.mode = modeList
	default:
		if key.Text != "" {
			m.createTitle += key.Text
		}
	}
	return nil
}

// openDetail starts fetching the selected document's bytes for the detail view.
func (m *Model) openDetail() tea.Cmd {
	visible := m.visible()
	if m.cursor < 0 || m.cursor >= len(visible) {
		return nil
	}
	ref := visible[m.cursor].ID
	m.detailRef = ref
	m.detail = ""
	m.mode = modeDetail
	store := m.store
	return func() tea.Msg {
		if store == nil {
			return detailMsg{ref: ref}
		}
		found, err := store.Find(ref)
		if err != nil {
			return detailMsg{ref: ref, err: err}
		}
		return detailMsg{ref: ref, content: string(found.Data)}
	}
}

// visible returns the loaded documents that pass the current filter, in the
// order the store returned them. It reads nothing from disk.
func (m *Model) visible() []vault.IndexRecord {
	if m.filter == (Filter{}) {
		return m.records
	}
	out := make([]vault.IndexRecord, 0, len(m.records))
	for _, record := range m.records {
		if m.filter.matches(record) {
			out = append(out, record)
		}
	}
	return out
}

// move shifts the selection by delta, clamped to the visible list.
func (m *Model) move(delta int) {
	m.cursor += delta
	m.clampCursor()
}

// clampCursor keeps the selection inside the visible list.
func (m *Model) clampCursor() {
	n := len(m.visible())
	switch {
	case n == 0:
		m.cursor = 0
	case m.cursor >= n:
		m.cursor = n - 1
	case m.cursor < 0:
		m.cursor = 0
	}
}

// recordFor returns the loaded record with the given reference.
func (m *Model) recordFor(ref string) (vault.IndexRecord, bool) {
	for _, record := range m.records {
		if record.ID == ref {
			return record, true
		}
	}
	return vault.IndexRecord{}, false
}

// listRows is how many document rows fit between the header and the footer.
func (m *Model) listRows() int {
	if m.height <= 3 {
		return 1
	}
	return m.height - 3
}

func (m *Model) renderList() string {
	visible := m.visible()
	header := fmt.Sprintf("specshelf — %d shown of %d documents", len(visible), len(m.records))
	if len(m.warnings) > 0 {
		header += fmt.Sprintf(", %d unreadable", len(m.warnings))
	}
	lines := []string{header}
	if summary := m.filter.summary(); summary != "" {
		lines = append(lines, "filter: "+summary)
	}
	if m.err != nil {
		lines = append(lines, "error: "+m.err.Error())
	}
	if len(visible) == 0 {
		lines = append(lines, "  no matching documents")
	}
	titleWidth := m.width - 40
	if titleWidth < 10 {
		titleWidth = 10
	}
	for i, record := range visible {
		if i >= m.listRows() {
			break
		}
		marker := " "
		if i == m.cursor {
			marker = ">"
		}
		parts := []string{
			marker + " " + record.Kind,
			fit(record.Title, titleWidth),
			record.Status,
			record.Updated,
			strings.Join(record.Tags, ","),
		}
		lines = append(lines, strings.Join(parts, "  "))
	}
	lines = append(lines, "j/k move · enter open · / filter · c create · ? help · q quit")
	return frame(lines, m.width)
}

func (m *Model) renderDetail() string {
	lines := []string{m.detailRef}
	if record, ok := m.recordFor(m.detailRef); ok {
		lines[0] = record.Title
		lines = append(lines, fmt.Sprintf("%s · %s · updated %s · %s", record.Kind, record.Status, record.Updated, record.Path))
	}
	lines = append(lines, strings.Repeat("─", min(m.width, 40)))
	room := m.height - 5
	if room < 1 {
		room = 1
	}
	body := strings.Split(m.detail, "\n")
	for i, line := range body {
		if i >= room {
			break
		}
		lines = append(lines, line)
	}
	lines = append(lines, "esc back · q quit")
	return frame(lines, m.width)
}

func (m *Model) renderCreate() string {
	return frame([]string{
		"New document",
		"",
		"kind:  " + m.createKind + "  (tab to switch)",
		"title: " + m.createTitle + "_",
		"",
		"enter create · esc cancel · q quit",
	}, m.width)
}

func (m *Model) renderHelp() string {
	lines := []string{"specshelf help", ""}
	for _, binding := range Bindings {
		lines = append(lines, binding.keyLine())
	}
	lines = append(lines, "", "press esc to go back, q to quit")
	return frame(lines, m.width)
}

// frame fits every line to the terminal width and joins them.
func frame(lines []string, width int) string {
	for i, line := range lines {
		lines[i] = fit(line, width)
	}
	return strings.Join(lines, "\n")
}

// fit truncates s to at most width runes, ending an over-long line with an
// ellipsis. It measures runes, not display columns, so a line of double-width
// characters can still overflow visually.
//
// shortcut: rune-count truncation, not display width; upgrade if titles from
// CJK vaults start breaking the list alignment.
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width == 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "…"
}

// dropLastRune removes the last rune from s, for backspace in a text prompt.
func dropLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}
