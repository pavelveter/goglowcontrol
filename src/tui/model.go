package tui

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusArea int

const (
	focusAliases focusArea = iota
	focusScenes
)

type subFocusArea int

const (
	subColors subFocusArea = iota
	subTemps
	subBrightness
)

type commandResultMsg struct {
	message string
	err     error
}

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	sectionStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	boxStyle       = lipgloss.NewStyle().Border(lipgloss.HiddenBorder()).Padding(0, 1)
	statusOkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	selectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true)
	focusedTitle   = sectionStyle.Copy().Underline(true)
	checkStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	aliasActions   = []string{"on", "off"}
	mainHelpText   = "Tab switch lists • Up/Down move • Left/Right choose alias on/off • Enter/Space run selection • Ctrl+C/Esc/q quit"
	subHelpText    = "Tab switch panels • Arrows move • Enter/Space apply • Backspace return • Ctrl+C/Esc/q quit"
	selectHelpText = "Up/Down move • Enter/Space toggle ✓ • Backspace return • Ctrl+C/Esc/q quit"
)

// model holds UI state.
type model struct {
	deps          Deps
	status        string
	submitting    bool
	colors        []string
	aliases       []string
	scenes        []string
	width         int
	listFocus     focusArea
	aliasIndex    int
	sceneIndex    int
	actionIndex   int
	statusFromLog bool
	actionFocus   bool
	inSubmenu     bool
	submenuAlias  string
	subFocus      subFocusArea
	colorCols     int
	colorIndex    int
	tempIndex     int
	brightIndex   int
	state         *State
	selecting     bool
	selectIndex   int
	selected      map[string]bool
	logChan       <-chan string
}

// newModel builds the Bubble Tea model with initial data and listeners.
func newModel(deps Deps, logCh <-chan string) model {
	deps.defaults()
	state := deps.State
	if state == nil {
		state = NewState(deps.MinBrightness, deps.MaxBrightness, deps.MinTemperature, deps.MaxTemperature)
		deps.State = state
	}

	m := model{
		deps:    deps,
		status:  "Use alias/scenes blocks or enter targets for future commands",
		colors:  deps.Colors,
		aliases: deps.Aliases,
		scenes:  deps.Scenes,
		state:   state,
		selected: func() map[string]bool {
			return make(map[string]bool)
		}(),
		logChan: logCh,
		colorCols: func() int {
			return computeColorCols(96)
		}(),
		listFocus: func() focusArea {
			if len(deps.Aliases) == 0 && len(deps.Scenes) > 0 {
				return focusScenes
			}
			return focusAliases
		}(),
	}
	return m
}

// Init starts background listeners.
func (m model) Init() tea.Cmd {
	return m.listenForLog()
}

// Update handles all incoming Bubble Tea messages (keys, mouse, resize, logs).
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.colorCols = computeColorCols(msg.Width)
		m.realignSubmenuSelections()
		m.realignSelector()
		return m, nil
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if msg.Y == 1 && m.clickedQuit(msg.X) {
			if m.selecting {
				m.exitSelector()
				return m, nil
			}
			if m.inSubmenu {
				m.exitSubmenu()
				return m, nil
			}
			return m, tea.Quit
		}
		return m, m.handleMouse(msg)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.selecting {
				m.exitSelector()
				return m, nil
			}
			if m.inSubmenu {
				m.exitSubmenu()
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyBackspace, tea.KeyCtrlH:
			if m.selecting {
				m.exitSelector()
				return m, nil
			}
			if m.inSubmenu {
				m.exitSubmenu()
				return m, nil
			}
			return m, nil
		case tea.KeyRunes:
			switch strings.ToLower(msg.String()) {
			case "q", "й":
				if m.selecting {
					m.exitSelector()
					return m, nil
				}
				if m.inSubmenu {
					m.exitSubmenu()
					return m, nil
				}
				return m, tea.Quit
			}
		case tea.KeyTab:
			if m.selecting {
				return m, nil
			}
			if m.inSubmenu {
				m.subFocus = (m.subFocus + 1) % 3
				return m, nil
			}
			m.toggleListFocus()
			return m, nil
		case tea.KeyShiftTab:
			if m.selecting {
				return m, nil
			}
			if m.inSubmenu {
				m.subFocus = (m.subFocus + 3 - 1) % 3
				return m, nil
			}
			m.toggleListFocus()
			return m, nil
		case tea.KeyUp:
			if m.selecting {
				m.moveSelector(-1)
				return m, nil
			}
			if m.inSubmenu {
				m.moveSubmenuSelection(-1, 0)
				return m, nil
			}
			m.moveListSelection(-1)
			return m, nil
		case tea.KeyDown:
			if m.selecting {
				m.moveSelector(1)
				return m, nil
			}
			if m.inSubmenu {
				m.moveSubmenuSelection(1, 0)
				return m, nil
			}
			m.moveListSelection(1)
			return m, nil
		case tea.KeyLeft:
			if m.selecting {
				return m, nil
			}
			if m.inSubmenu {
				m.moveSubmenuSelection(0, -1)
				return m, nil
			}
			if m.listFocus == focusAliases {
				if !m.actionFocus {
					m.actionFocus = true
					m.actionIndex = len(aliasActions) - 1
					return m, nil
				}
				if m.actionIndex == 0 {
					m.actionFocus = false
					return m, nil
				}
				m.actionIndex = (m.actionIndex + len(aliasActions) - 1) % len(aliasActions)
				return m, nil
			}
		case tea.KeyRight:
			if m.selecting {
				return m, nil
			}
			if m.inSubmenu {
				m.moveSubmenuSelection(0, 1)
				return m, nil
			}
			if m.listFocus == focusAliases {
				if !m.actionFocus {
					m.actionFocus = true
					m.actionIndex = 0
					return m, nil
				}
				if m.actionIndex == len(aliasActions)-1 {
					m.actionFocus = false
					return m, nil
				}
				m.actionIndex = (m.actionIndex + 1) % len(aliasActions)
				return m, nil
			}
		case tea.KeyEnter, tea.KeySpace:
			if m.selecting {
				m.toggleSelector()
				return m, nil
			}
			if m.inSubmenu {
				return m, m.applySubmenuSelection()
			}
			if m.listFocus == focusScenes {
				if m.sceneIndex == len(m.scenes) {
					m.enterSelector()
					return m, nil
				}
				if len(m.scenes) > 0 {
					scene := m.scenes[m.sceneIndex]
					m.status = fmt.Sprintf("Running scene %s...", scene)
					m.submitting = true
					return m, m.runScene(scene)
				}
			}
			if m.listFocus == focusAliases && len(m.aliases) > 0 && m.actionFocus {
				alias := m.aliases[m.aliasIndex]
				action := aliasActions[m.actionIndex%len(aliasActions)]
				m.status = fmt.Sprintf("Sending %s to %s...", action, alias)
				m.submitting = true
				m.statusFromLog = false
				return m, m.runAliasAction(alias, action)
			}
			if m.listFocus == focusAliases && len(m.aliases) > 0 && !m.actionFocus {
				m.enterSubmenu(m.aliases[m.aliasIndex])
				return m, nil
			}
			if m.submitting {
				return m, nil
			}
			m.status = "Sending command..."
			m.submitting = true
			return m, m.runCommand()
		}
	case logLineMsg:
		if msg.line != "" && m.submitting {
			m.status = statusErrStyle.Render(cleanStatus(msg.line))
			m.statusFromLog = true
		}
		return m, m.listenForLog()
	case commandResultMsg:
		m.submitting = false
		m.statusFromLog = false
		if msg.err != nil {
			m.status = statusErrStyle.Render(cleanStatus(fmt.Sprintf("Error: %v", msg.err)))
		} else {
			m.status = statusOkStyle.Render(cleanStatus(msg.message))
		}
		return m, m.listenForLog()
	}

	return m, nil
}

// toggleListFocus swaps focus between aliases and scenes.
func (m *model) toggleListFocus() {
	if m.listFocus == focusAliases {
		m.listFocus = focusScenes
	} else {
		m.listFocus = focusAliases
	}
	m.actionFocus = false
	m.inSubmenu = false
	m.selecting = false
}

// moveListSelection moves selection in the active list.
func (m *model) moveListSelection(delta int) {
	switch m.listFocus {
	case focusAliases:
		if len(m.aliases) == 0 {
			return
		}
		m.aliasIndex = (m.aliasIndex + delta + len(m.aliases)) % len(m.aliases)
		m.actionFocus = false
		m.inSubmenu = false
	case focusScenes:
		count := m.sceneItemCount()
		if count == 0 {
			return
		}
		m.sceneIndex = (m.sceneIndex + delta + count) % count
	}
}

// sceneItemCount returns number of rows in the scenes list (scenes + add button).
func (m model) sceneItemCount() int {
	return len(m.scenes) + 1
}

// runCommand is a placeholder since TUI uses direct alias operations.
func (m model) runCommand() tea.Cmd {
	return func() tea.Msg {
		return commandResultMsg{err: errors.New("no command inputs available in TUI")}
	}
}

// enterSelector opens alias selection view.
func (m *model) enterSelector() {
	m.selecting = true
	m.selectIndex = 0
}

// exitSelector closes alias selection view.
func (m *model) exitSelector() {
	m.selecting = false
}

// moveSelector moves selection in alias selector.
func (m *model) moveSelector(delta int) {
	if len(m.aliases) == 0 {
		return
	}
	m.selectIndex = (m.selectIndex + delta + len(m.aliases)) % len(m.aliases)
}

// toggleSelector toggles current alias selection.
func (m *model) toggleSelector() {
	if len(m.aliases) == 0 {
		return
	}
	alias := m.aliases[m.selectIndex]
	m.selected[alias] = !m.selected[alias]
}

// realignSelector clamps selector index on data change/resize.
func (m *model) realignSelector() {
	m.selectIndex = clampIndex(m.selectIndex, len(m.aliases))
}

// enterSubmenu opens the color/temp/brightness submenu for the alias.
func (m *model) enterSubmenu(alias string) {
	m.inSubmenu = true
	m.submenuAlias = alias
	m.subFocus = subColors
	m.actionFocus = false
	if m.colorCols == 0 {
		m.colorCols = computeColorCols(m.width)
	}
	m.colorIndex = 0
	m.tempIndex = 0
	m.brightIndex = 0
}

// exitSubmenu closes the submenu and resets related state.
func (m *model) exitSubmenu() {
	m.inSubmenu = false
	m.submenuAlias = ""
	m.subFocus = subColors
	m.actionFocus = false
}

// realignSubmenuSelections bounds submenu indices on resize/data change.
func (m *model) realignSubmenuSelections() {
	if m.colorCols <= 0 {
		m.colorCols = computeColorCols(m.width)
	}
	m.colorIndex = clampIndex(m.colorIndex, len(m.colors))
	m.tempIndex = clampIndex(m.tempIndex, m.deps.ValueSteps)
	m.brightIndex = clampIndex(m.brightIndex, m.deps.ValueSteps)
}

// moveSubmenuSelection updates selection inside the submenu panels.
func (m *model) moveSubmenuSelection(deltaRow, deltaCol int) {
	switch m.subFocus {
	case subColors:
		if len(m.colors) == 0 {
			return
		}
		width := m.colorContentWidth()
		counts := layoutWords(m.colors, width)
		m.colorIndex = moveInLines(m.colorIndex, counts, deltaRow, deltaCol)
	case subTemps:
		count := m.deps.ValueSteps
		m.tempIndex = clampIndex(m.tempIndex+deltaRow+deltaCol, count)
	case subBrightness:
		count := m.deps.ValueSteps
		m.brightIndex = clampIndex(m.brightIndex+deltaRow+deltaCol, count)
	}
}

// applySubmenuSelection executes the currently highlighted submenu option.
func (m model) applySubmenuSelection() tea.Cmd {
	if !m.inSubmenu || m.submenuAlias == "" {
		return nil
	}
	switch m.subFocus {
	case subColors:
		if len(m.colors) == 0 {
			return nil
		}
		color := m.colors[m.colorIndex%len(m.colors)]
		m.status = fmt.Sprintf("Applying color %s to %s...", color, m.submenuAlias)
		m.submitting = true
		return m.runAliasCommand(m.submenuAlias, "color", color)
	case subTemps:
		tempValues := buildRange(m.deps.MinTemperature, m.deps.MaxTemperature, m.deps.ValueSteps)
		temp := tempValues[m.tempIndex%len(tempValues)]
		m.status = fmt.Sprintf("Applying temp %d to %s...", temp, m.submenuAlias)
		m.submitting = true
		return m.runAliasCommand(m.submenuAlias, "t", fmt.Sprintf("%d", temp))
	case subBrightness:
		brightValues := buildRange(m.deps.MinBrightness, m.deps.MaxBrightness, m.deps.ValueSteps)
		bright := brightValues[m.brightIndex%len(brightValues)]
		m.status = fmt.Sprintf("Applying brightness %d to %s...", bright, m.submenuAlias)
		m.submitting = true
		return m.runAliasCommand(m.submenuAlias, "brightness", fmt.Sprintf("%d", bright))
	}
	return nil
}

// clickedQuit detects clicks on the "quit" word in the help line.
func (m model) clickedQuit(x int) bool {
	help := mainHelpText
	if m.selecting {
		help = selectHelpText
	}
	if m.inSubmenu {
		help = subHelpText
	}
	bytePos := strings.Index(help, "quit")
	if bytePos == -1 {
		return false
	}
	runeStart := utf8.RuneCountInString(help[:bytePos])
	runeEnd := runeStart + len([]rune("quit"))
	return x >= runeStart && x < runeEnd
}

// handleMouse dispatches mouse clicks to the appropriate handler.
func (m *model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if m.selecting {
		return m.handleMouseSelector(msg)
	}
	if m.inSubmenu {
		return m.handleMouseSubmenu(msg)
	}
	return m.handleMouseMain(msg)
}

// handleMouseMain reacts to clicks in the main two-panel view.
func (m *model) handleMouseMain(msg tea.MouseMsg) tea.Cmd {
	usableWidth := computeUsableWidth(m.width)
	leftWidth := usableWidth / 2
	rightWidth := usableWidth - leftWidth
	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 20 {
		rightWidth = 20
	}
	top := 2 // header + help

	if msg.Y < top {
		return nil
	}

	// Aliases panel
	if msg.X < leftWidth {
		line := msg.Y - top
		if line == 0 {
			return nil
		}
		idx := line - 1
		if idx < 0 || idx >= len(m.aliases) {
			return nil
		}
		m.listFocus = focusAliases
		m.aliasIndex = idx
		m.inSubmenu = false
		m.actionFocus = false

		// Detect click on actions for active alias line.
		contentX := msg.X - 1
		if idx == m.aliasIndex && len(aliasActions) > 0 {
			actionStart := len(m.aliases[idx]) + 3 // "> "+alias+" "
			if contentX >= actionStart {
				m.actionFocus = true
				if contentX < actionStart+len(aliasActions[0])+2 {
					m.actionIndex = 0
				} else {
					m.actionIndex = 1 % len(aliasActions)
				}
				alias := m.aliases[m.aliasIndex]
				action := aliasActions[m.actionIndex]
				m.status = fmt.Sprintf("Sending %s to %s...", action, alias)
				m.submitting = true
				return m.runAliasAction(alias, action)
			}
		}
		// Click on alias body opens submenu
		m.enterSubmenu(m.aliases[m.aliasIndex])
		return nil
	}

	// Scenes panel
	if msg.X >= leftWidth && msg.X < leftWidth+rightWidth {
		line := msg.Y - top
		if line == 0 {
			return nil
		}
		idx := line - 1
		total := m.sceneItemCount()
		if idx < 0 || idx >= total {
			return nil
		}
		m.listFocus = focusScenes
		m.sceneIndex = idx
		if idx == len(m.scenes) {
			m.enterSelector()
			return nil
		}
		scene := m.scenes[m.sceneIndex]
		m.status = fmt.Sprintf("Running scene %s...", scene)
		m.submitting = true
		return m.runScene(scene)
	}

	return nil
}

// handleMouseSelector reacts to clicks inside alias selector view.
func (m *model) handleMouseSelector(msg tea.MouseMsg) tea.Cmd {
	top := 2 // header + help
	if msg.Y < top {
		return nil
	}
	line := msg.Y - top
	if line == 0 {
		return nil
	}
	idx := line - 1
	if idx < 0 || idx >= len(m.aliases) {
		return nil
	}
	m.selectIndex = idx
	m.toggleSelector()
	return nil
}

// handleMouseSubmenu reacts to clicks inside the submenu panels.
func (m *model) handleMouseSubmenu(msg tea.MouseMsg) tea.Cmd {
	usableWidth := computeUsableWidth(m.width)
	colWidth, tempWidth, brightWidth := computeSubmenuWidths(usableWidth)
	top := 2 // header + help

	if msg.Y < top {
		return nil
	}

	// Colors panel region
	if msg.X < colWidth {
		line := msg.Y - top
		if line == 0 {
			return nil
		}
		contentX := msg.X - 1
		lines := layoutWords(m.colors, m.colorContentWidth())
		lineIdx := line - 1
		if lineIdx < 0 || lineIdx >= len(lines) {
			return nil
		}
		start := 0
		for i := 0; i < lineIdx; i++ {
			start += lines[i]
		}
		end := start + lines[lineIdx]
		if end > len(m.colors) {
			end = len(m.colors)
		}
		words := m.colors[start:end]
		wordIdx := pickWordIndex(words, contentX)
		m.subFocus = subColors
		m.colorIndex = start + wordIdx
		return m.applySubmenuSelection()
	}

	// Temperature panel
	if msg.X >= colWidth && msg.X < colWidth+tempWidth {
		line := msg.Y - top - 1
		if line < 0 || line >= m.deps.ValueSteps {
			return nil
		}
		m.subFocus = subTemps
		m.tempIndex = line
		return m.applySubmenuSelection()
	}

	// Brightness panel
	if msg.X >= colWidth+tempWidth && msg.X < colWidth+tempWidth+brightWidth {
		line := msg.Y - top - 1
		if line < 0 || line >= m.deps.ValueSteps {
			return nil
		}
		m.subFocus = subBrightness
		m.brightIndex = line
		return m.applySubmenuSelection()
	}

	return nil
}

// runAliasAction sends an on/off style action to all IPs of an alias.
func (m model) runAliasAction(alias, action string) tea.Cmd {
	return func() tea.Msg {
		if m.deps.ResolveTargets == nil || m.deps.Execute == nil {
			return commandResultMsg{err: errors.New("no alias action handler")}
		}
		ips, err := m.deps.ResolveTargets([]string{alias})
		if err != nil {
			return commandResultMsg{err: err}
		}
		if len(ips) == 0 {
			return commandResultMsg{err: fmt.Errorf("alias %s has no IPs", alias)}
		}
		if m.state != nil {
			m.state.RecordAction(alias, action)
		}
		var wg sync.WaitGroup
		for _, ip := range ips {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				m.deps.Execute(ip, action, "")
			}(ip)
		}
		wg.Wait()
		return commandResultMsg{
			message: fmt.Sprintf("Sent %s to %s (%d target(s))", action, alias, len(ips)),
		}
	}
}

// runAliasCommand sends a command with parameter to all IPs of an alias.
func (m model) runAliasCommand(alias, command, param string) tea.Cmd {
	return func() tea.Msg {
		if m.deps.ResolveTargets == nil || m.deps.Execute == nil {
			return commandResultMsg{err: errors.New("no alias command handler")}
		}
		ips, err := m.deps.ResolveTargets([]string{alias})
		if err != nil {
			return commandResultMsg{err: err}
		}
		if len(ips) == 0 {
			return commandResultMsg{err: fmt.Errorf("alias %s has no IPs", alias)}
		}
		if m.state != nil {
			m.state.RecordCommand(alias, command, param)
		}
		var wg sync.WaitGroup
		for _, ip := range ips {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				m.deps.Execute(ip, command, param)
			}(ip)
		}
		wg.Wait()
		return commandResultMsg{
			message: fmt.Sprintf("Sent %s %s to %s (%d target(s))", command, param, alias, len(ips)),
		}
	}
}

// runScene triggers a scene callback.
func (m model) runScene(sceneName string) tea.Cmd {
	return func() tea.Msg {
		if m.deps.RunScene == nil {
			return commandResultMsg{err: errors.New("no scene handler")}
		}
		if m.state != nil && m.deps.SceneCommands != nil {
			m.state.RecordScene(m.deps.SceneCommands[sceneName])
		}
		msg, err := m.deps.RunScene(sceneName)
		if msg == "" {
			msg = fmt.Sprintf("Scene %s executed", sceneName)
		}
		return commandResultMsg{message: msg, err: err}
	}
}

// listenForLog converts log channel messages into Bubble Tea messages.
func (m model) listenForLog() tea.Cmd {
	if m.logChan == nil {
		return nil
	}
	return func() tea.Msg {
		line, ok := <-m.logChan
		if !ok {
			return nil
		}
		return logLineMsg{line: line}
	}
}
