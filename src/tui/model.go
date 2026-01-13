package tui

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
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

type sceneSavedMsg struct {
	name string
	err  error
	// updated is true when an existing line was replaced.
	updated  bool
	commands []string
}

type addFocusArea int

const (
	addFocusName addFocusArea = iota
	addFocusAliases
	addFocusButtons
)

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	sectionStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	boxStyle       = lipgloss.NewStyle().Border(lipgloss.HiddenBorder()).Padding(0, 1)
	statusOkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	selectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true)
	focusedTitle   = sectionStyle.Copy().Underline(true)
	aliasActions   = []string{"on", "off"}
	mainHelpText   = "Tab switch lists • Up/Down move • Left/Right choose alias on/off • Enter/Space run selection • Ctrl+C/Esc/q quit"
	subHelpText    = "Tab switch panels • Arrows move • Enter/Space apply • Backspace return • Ctrl+C/Esc/q quit"
	addSceneHelp   = "Tab Name/Aliases/Buttons • Enter/Space toggle/select • Esc/Ctrl+C/q quit"
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
	addingScene   bool
	addMenuIndex  int
	addFocus      addFocusArea
	addAliasIndex int
	addSelected   map[string]bool
	sceneName     textinput.Model
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

	nameInput := textinput.New()
	nameInput.Placeholder = "Scene name"
	nameInput.CharLimit = 64
	nameInput.Prompt = "Name: "
	nameInput.Width = 40
	nameInput.Blur()

	m := model{
		deps:    deps,
		status:  "Use alias/scenes blocks or enter targets for future commands",
		colors:  deps.Colors,
		aliases: deps.Aliases,
		scenes:  deps.Scenes,
		state:   state,
		logChan: logCh,
		sceneName: func() textinput.Model {
			return nameInput
		}(),
		addSelected: func() map[string]bool {
			return make(map[string]bool)
		}(),
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
		m.realignAddMenu()
		return m, nil
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if msg.Y == 1 && m.clickedQuit(msg.X) {
			if m.addingScene {
				m.exitAddScene()
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
		if m.addingScene {
			return m.handleAddSceneKeys(msg)
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.inSubmenu {
				m.exitSubmenu()
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyBackspace, tea.KeyCtrlH:
			if m.inSubmenu {
				m.exitSubmenu()
				return m, nil
			}
			return m, nil
		case tea.KeyRunes:
			switch strings.ToLower(msg.String()) {
			case "q", "й":
				if m.inSubmenu {
					m.exitSubmenu()
					return m, nil
				}
				return m, tea.Quit
			}
		case tea.KeyTab:
			if m.inSubmenu {
				m.subFocus = (m.subFocus + 1) % 3
				return m, nil
			}
			m.toggleListFocus()
			return m, nil
		case tea.KeyShiftTab:
			if m.inSubmenu {
				m.subFocus = (m.subFocus + 3 - 1) % 3
				return m, nil
			}
			m.toggleListFocus()
			return m, nil
		case tea.KeyUp:
			if m.inSubmenu {
				m.moveSubmenuSelection(-1, 0)
				return m, nil
			}
			m.moveListSelection(-1)
			return m, nil
		case tea.KeyDown:
			if m.inSubmenu {
				m.moveSubmenuSelection(1, 0)
				return m, nil
			}
			m.moveListSelection(1)
			return m, nil
		case tea.KeyLeft:
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
			if m.inSubmenu {
				return m, m.applySubmenuSelection()
			}
			if m.listFocus == focusScenes {
				if m.sceneIndex == len(m.scenes) {
					m.openAddSceneMenu()
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
	case sceneSavedMsg:
		m.submitting = false
		if msg.err != nil {
			m.status = statusErrStyle.Render(cleanStatus(fmt.Sprintf("Error: %v", msg.err)))
			m.setAddFocus(addFocusName)
			return m, m.listenForLog()
		}
		if msg.updated {
			m.status = statusOkStyle.Render(cleanStatus(fmt.Sprintf("Scene %s updated", msg.name)))
		} else {
			m.status = statusOkStyle.Render(cleanStatus(fmt.Sprintf("Scene %s saved", msg.name)))
		}
		m.exitAddScene()
		if idx := m.findSceneIndex(msg.name); idx >= 0 {
			m.sceneIndex = idx
		} else {
			m.scenes = append(m.scenes, msg.name)
			m.sceneIndex = len(m.scenes) - 1
		}
		m.listFocus = focusScenes
		m.actionFocus = false
		if m.deps.SceneCommands != nil {
			m.deps.SceneCommands[msg.name] = msg.commands
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
	m.addingScene = false
	m.addSelected = make(map[string]bool)
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

// sceneExists checks if a scene with the given name is already listed.
func (m model) sceneExists(name string) bool {
	name = strings.TrimSpace(name)
	for _, scene := range m.scenes {
		if scene == name {
			return true
		}
	}
	return false
}

// findSceneIndex returns index of scene by name or -1.
func (m model) findSceneIndex(name string) int {
	for i, scene := range m.scenes {
		if scene == name {
			return i
		}
	}
	return -1
}

// runCommand is a placeholder since TUI uses direct alias operations.
func (m model) runCommand() tea.Cmd {
	return func() tea.Msg {
		return commandResultMsg{err: errors.New("no command inputs available in TUI")}
	}
}

// openAddSceneMenu opens the add scene menu.
func (m *model) openAddSceneMenu() {
	m.addingScene = true
	m.addMenuIndex = 0
	m.addFocus = addFocusName
	m.addAliasIndex = 0
	m.addSelected = make(map[string]bool)
	m.sceneName.SetValue("")
	m.sceneName.CursorEnd()
	m.sceneName.Focus()
}

// exitAddScene closes the add scene flow.
func (m *model) exitAddScene() {
	m.addingScene = false
	m.addMenuIndex = 0
	m.sceneName.SetValue("")
	m.sceneName.Blur()
	m.addSelected = make(map[string]bool)
	m.addFocus = addFocusName
	m.addAliasIndex = 0
}

// realignAddMenu clamps menu selection on resize.
func (m *model) realignAddMenu() {
	m.addMenuIndex = clampIndex(m.addMenuIndex, 2)
	m.addAliasIndex = clampIndex(m.addAliasIndex, len(m.aliases))
}

// setAddFocus switches focus for add-scene mode and keeps the text input synced.
func (m *model) setAddFocus(target addFocusArea) {
	m.addFocus = target
	if target == addFocusName {
		m.sceneName.Focus()
	} else {
		m.sceneName.Blur()
	}
}

// cycleAddFocus rotates focus between name, aliases, and buttons.
func (m *model) cycleAddFocus(delta int) {
	current := int(m.addFocus)
	for i := 0; i < 3; i++ {
		next := (current + delta + 3) % 3
		if next == int(addFocusAliases) && len(m.aliases) == 0 {
			current = next
			continue
		}
		m.setAddFocus(addFocusArea(next))
		return
	}
}

// toggleAliasSelection flips selection state for the alias at index.
func (m *model) toggleAliasSelection(idx int) {
	if idx < 0 || idx >= len(m.aliases) {
		return
	}
	alias := m.aliases[idx]
	m.addSelected[alias] = !m.addSelected[alias]
}

// buildSceneCommands constructs commands for selected aliases (default "on").
func (m model) buildSceneCommands() []string {
	var cmds []string
	for _, alias := range m.aliases {
		if m.addSelected[alias] {
			if cmd, ok := m.buildSceneCommand(alias); ok {
				cmds = append(cmds, cmd)
			}
		}
	}
	return cmds
}

// buildSceneCommand creates a single scene command using current state, falling back to on.
func (m model) buildSceneCommand(alias string) (string, bool) {
	if m.state == nil {
		return fmt.Sprintf("%s on", alias), true
	}
	snap := m.state.Snapshot()
	state, ok := snap[alias]
	if !ok {
		return fmt.Sprintf("%s on", alias), true
	}
	// Prefer explicit off.
	if strings.ToLower(state.Power) == "off" {
		return fmt.Sprintf("%s off", alias), true
	}
	// Prefer color if set.
	if state.Color != "" {
		return fmt.Sprintf("%s %s", alias, state.Color), true
	}
	// Then brightness.
	if state.Brightness != 0 {
		return fmt.Sprintf("%s brightness %d", alias, state.Brightness), true
	}
	// Then temperature.
	if state.Temperature != 0 {
		return fmt.Sprintf("%s t %d", alias, state.Temperature), true
	}
	// Fallback to power on.
	if state.Power != "" {
		return fmt.Sprintf("%s %s", alias, state.Power), true
	}
	return fmt.Sprintf("%s on", alias), true
}

// handleAddSceneKeys manages key handling inside the add scene views.
func (m model) handleAddSceneKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.submitting {
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.exitAddScene()
		return m, nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		if m.addFocus == addFocusName {
			var cmd tea.Cmd
			m.sceneName, cmd = m.sceneName.Update(msg)
			return m, cmd
		}
		m.exitAddScene()
		return m, nil
	case tea.KeyTab:
		m.cycleAddFocus(1)
		return m, nil
	case tea.KeyShiftTab:
		m.cycleAddFocus(-1)
		return m, nil
	}

	switch m.addFocus {
	case addFocusName:
		var cmd tea.Cmd
		m.sceneName, cmd = m.sceneName.Update(msg)
		if msg.Type == tea.KeyEnter {
			m.cycleAddFocus(1)
			return m, nil
		}
		return m, cmd
	case addFocusAliases:
		if len(m.aliases) == 0 {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyUp:
			m.addAliasIndex = (m.addAliasIndex + len(m.aliases) - 1) % len(m.aliases)
			return m, nil
		case tea.KeyDown:
			m.addAliasIndex = (m.addAliasIndex + 1) % len(m.aliases)
			return m, nil
		case tea.KeyEnter, tea.KeySpace:
			m.toggleAliasSelection(m.addAliasIndex)
			return m, nil
		}
	case addFocusButtons:
		switch msg.Type {
		case tea.KeyLeft, tea.KeyRight:
			m.addMenuIndex = (m.addMenuIndex + 1) % 2
			return m, nil
		case tea.KeyEnter, tea.KeySpace:
			return m, m.applyAddMenuSelection()
		}
	}
	return m, nil
}

// applyAddMenuSelection runs the highlighted menu action.
func (m *model) applyAddMenuSelection() tea.Cmd {
	switch m.addMenuIndex {
	case 0: // Add
		name := strings.TrimSpace(m.sceneName.Value())
		if name == "" {
			m.status = statusErrStyle.Render("Scene name cannot be empty")
			m.setAddFocus(addFocusName)
			return nil
		}
		cmds := m.buildSceneCommands()
		if len(cmds) == 0 {
			m.status = statusErrStyle.Render("Select at least one alias")
			m.setAddFocus(addFocusAliases)
			return nil
		}
		if m.sceneExists(name) {
			m.status = fmt.Sprintf("Updating scene %s...", name)
		} else {
			m.status = fmt.Sprintf("Saving scene %s...", name)
		}
		m.submitting = true
		return m.saveScene(name, cmds)
	default: // Cancel
		m.exitAddScene()
		return nil
	}
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
	if m.addingScene {
		help = addSceneHelp
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
	if m.addingScene {
		return m.handleMouseAddScene(msg)
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
			m.openAddSceneMenu()
			return nil
		}
		scene := m.scenes[m.sceneIndex]
		m.status = fmt.Sprintf("Running scene %s...", scene)
		m.submitting = true
		return m.runScene(scene)
	}

	return nil
}

// handleMouseAddScene reacts to clicks inside add scene views.
func (m *model) handleMouseAddScene(msg tea.MouseMsg) tea.Cmd {
	usableWidth := computeUsableWidth(m.width)
	top := 2 // header + help
	if msg.Y < top {
		return nil
	}

	nameStart := top
	aliasStart := nameStart + 2
	aliasLines := 1
	if len(m.aliases) > 0 {
		aliasLines += len(m.aliases)
	} else {
		aliasLines++
	}
	buttonStart := aliasStart + aliasLines
	buttonLine := buttonStart + 1

	if msg.Y == nameStart+1 {
		m.setAddFocus(addFocusName)
		m.sceneName.Focus()
		return nil
	}

	if msg.Y > aliasStart && msg.Y <= aliasStart+aliasLines-1 {
		if len(m.aliases) == 0 {
			return nil
		}
		idx := msg.Y - aliasStart - 1
		if idx >= 0 && idx < len(m.aliases) {
			m.setAddFocus(addFocusAliases)
			m.addAliasIndex = idx
			m.toggleAliasSelection(idx)
			return nil
		}
	}

	if msg.Y == buttonLine {
		m.setAddFocus(addFocusButtons)
		if msg.X < usableWidth/2 {
			m.addMenuIndex = 0
		} else {
			m.addMenuIndex = 1
		}
		return m.applyAddMenuSelection()
	}

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

// saveScene delegates scene persistence.
func (m model) saveScene(name string, commands []string) tea.Cmd {
	clean := strings.TrimSpace(name)
	return func() tea.Msg {
		if m.deps.SaveScene == nil {
			return sceneSavedMsg{name: clean, commands: commands, err: errors.New("no scene saver")}
		}
		updated, err := m.deps.SaveScene(clean, commands)
		return sceneSavedMsg{name: clean, commands: commands, err: err, updated: updated}
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
