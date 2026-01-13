package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the entire TUI frame depending on current mode.
func (m model) View() string {
	header := titleStyle.Render("Go GlowControl — Light TUI")
	help := helpStyle.Render(mainHelpText)

	totalWidth := m.width
	if totalWidth == 0 {
		totalWidth = 96
	}
	usableWidth := computeUsableWidth(totalWidth)

	if m.addingScene {
		header = titleStyle.Render("New scene")
		help = helpStyle.Render(addSceneHelp)
		layout := m.renderAddSceneLayout(usableWidth)
		statusBox := boxStyle.Width(usableWidth).Render(sectionStyle.Render("Status") + "\n" + m.status)
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			help,
			layout,
			statusBox,
		)
	}

	if m.inSubmenu {
		header = titleStyle.Render(m.submenuAlias)
		help = helpStyle.Render(subHelpText)

		colWidth, tempWidth, brightWidth := computeSubmenuWidths(usableWidth)

		panels := lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderColorsPanel(colWidth),
			m.renderTempsPanel(tempWidth),
			m.renderBrightPanel(brightWidth),
		)

		statusBox := boxStyle.Width(usableWidth).Render(sectionStyle.Render("Status") + "\n" + m.status)

		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			help,
			panels,
			statusBox,
		)
	}

	leftWidth := usableWidth / 2
	rightWidth := usableWidth - leftWidth
	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 20 {
		rightWidth = 20
	}

	lists := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderAliasBox(leftWidth),
		m.renderSceneBox(rightWidth),
	)

	statusBox := boxStyle.Width(usableWidth).Render(sectionStyle.Render("Status") + "\n" + m.status)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		help,
		lists,
		statusBox,
	)
}

// renderAliasBox renders the aliases panel with selection.
func (m model) renderAliasBox(width int) string {
	title := sectionStyle.Render("Aliases")
	if m.listFocus == focusAliases {
		title = focusedTitle.Render("Aliases")
	}
	body := m.renderAliasBody(m.listFocus == focusAliases)
	return boxStyle.Width(width).Render(title + "\n" + body)
}

// renderSceneBox renders the scenes panel with selection.
func (m model) renderSceneBox(width int) string {
	title := sectionStyle.Render("Scenes")
	if m.listFocus == focusScenes {
		title = focusedTitle.Render("Scenes")
	}
	body := m.renderSceneBodyWithAdd(m.listFocus == focusScenes)
	return boxStyle.Width(width).Render(title + "\n" + body)
}

// renderSceneBodyWithAdd shows scenes plus a trailing "+" item.
func (m model) renderSceneBodyWithAdd(active bool) string {
	total := m.sceneItemCount()
	if total == 0 {
		return "–"
	}
	var rows []string
	for i, scene := range m.scenes {
		if active && i == m.sceneIndex {
			rows = append(rows, selectedStyle.Render("> "+scene))
		} else {
			rows = append(rows, "  "+scene)
		}
	}
	addLabel := "+"
	if active && m.sceneIndex == len(m.scenes) {
		rows = append(rows, selectedStyle.Render("> "+addLabel))
	} else {
		rows = append(rows, "  "+addLabel)
	}
	return strings.Join(rows, "\n")
}

// renderAddSceneLayout shows name input, alias selection, and buttons.
func (m model) renderAddSceneLayout(width int) string {
	nameBox := m.renderAddSceneNameBox(width)
	aliasBox := m.renderAddSceneAliasBox(width)
	buttons := m.renderAddSceneButtons(width)
	return lipgloss.JoinVertical(lipgloss.Left, nameBox, aliasBox, buttons)
}

func (m model) renderAddSceneNameBox(width int) string {
	title := sectionStyle.Render("Scene Name")
	if m.addFocus == addFocusName {
		title = focusedTitle.Render("Scene Name")
	}
	return boxStyle.Width(width).Render(title + "\n" + m.sceneName.View())
}

func (m model) renderAddSceneAliasBox(width int) string {
	title := sectionStyle.Render("Aliases in Scene")
	if m.addFocus == addFocusAliases {
		title = focusedTitle.Render("Aliases in Scene")
	}
	if len(m.aliases) == 0 {
		return boxStyle.Width(width).Render(title + "\n–")
	}
	var rows []string
	for i, alias := range m.aliases {
		check := "[ ]"
		if m.addSelected[alias] {
			check = "[x]"
		}
		line := fmt.Sprintf("%s %s", check, alias)
		if m.state != nil {
			if summary, ok := m.state.Summary(alias); ok {
				line = fmt.Sprintf("%s  %s", line, summary)
			}
		}
		if m.addFocus == addFocusAliases && i == m.addAliasIndex {
			rows = append(rows, "> "+selectedStyle.Render(line))
		} else {
			rows = append(rows, "  "+line)
		}
	}
	return boxStyle.Width(width).Render(title + "\n" + strings.Join(rows, "\n"))
}

func (m model) renderAddSceneButtons(width int) string {
	title := sectionStyle.Render("Actions")
	if m.addFocus == addFocusButtons {
		title = focusedTitle.Render("Actions")
	}
	options := []string{"Add", "Cancel"}
	var rows []string
	for i, opt := range options {
		label := fmt.Sprintf("[ %s ]", opt)
		if m.addFocus == addFocusButtons && i == m.addMenuIndex {
			rows = append(rows, "> "+selectedStyle.Render(label))
		} else {
			rows = append(rows, "  "+label)
		}
	}
	return boxStyle.Width(width).Render(title + "\n" + strings.Join(rows, "\n"))
}

// renderColorsPanel draws the colors panel in submenu.
func (m model) renderColorsPanel(width int) string {
	title := sectionStyle.Render("Colors")
	if m.subFocus == subColors {
		title = focusedTitle.Render("Colors")
	}
	if len(m.colors) == 0 {
		return boxStyle.Width(width).Render(title + "\n–")
	}
	contentWidth := m.colorContentWidth()
	lines := layoutWords(m.colors, contentWidth)
	var b strings.Builder
	idx := 0
	for lineIdx, count := range lines {
		var parts []string
		for i := 0; i < count && idx < len(m.colors); i++ {
			name := m.colors[idx]
			display := name
			if m.deps.ColorHex != nil && m.deps.HexToRGB != nil {
				if hex, ok := m.deps.ColorHex(name); ok {
					display = m.deps.HexToRGB(hex) + name + m.deps.ColorReset
				}
			}
			if m.subFocus == subColors && idx == m.colorIndex {
				parts = append(parts, selectedStyle.Render(display))
			} else {
				parts = append(parts, display)
			}
			idx++
		}
		b.WriteString(strings.Join(parts, " "))
		if lineIdx < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return boxStyle.Width(width).Render(title + "\n" + b.String())
}

// renderTempsPanel draws the temperature values vertically.
func (m model) renderTempsPanel(width int) string {
	title := sectionStyle.Render("Temperature")
	if m.subFocus == subTemps {
		title = focusedTitle.Render("Temperature")
	}
	values := buildRange(m.deps.MinTemperature, m.deps.MaxTemperature, m.deps.ValueSteps)
	var b strings.Builder
	for i, v := range values {
		item := strconv.Itoa(v)
		if m.subFocus == subTemps && i == m.tempIndex {
			b.WriteString(selectedStyle.Render(item))
		} else {
			b.WriteString(item)
		}
		if i < len(values)-1 {
			b.WriteString("\n")
		}
	}
	return boxStyle.Width(width).Render(title + "\n" + b.String())
}

// renderBrightPanel draws the brightness values vertically.
func (m model) renderBrightPanel(width int) string {
	title := sectionStyle.Render("Brightness")
	if m.subFocus == subBrightness {
		title = focusedTitle.Render("Brightness")
	}
	values := buildRange(m.deps.MinBrightness, m.deps.MaxBrightness, m.deps.ValueSteps)
	var b strings.Builder
	for i, v := range values {
		item := strconv.Itoa(v)
		if m.subFocus == subBrightness && i == m.brightIndex {
			b.WriteString(selectedStyle.Render(item))
		} else {
			b.WriteString(item)
		}
		if i < len(values)-1 {
			b.WriteString("\n")
		}
	}
	return boxStyle.Width(width).Render(title + "\n" + b.String())
}

// renderListBody renders list items with optional selection highlighting.
// renderListBody renders list items with optional selection highlighting.
func renderListBody(items []string, selected int, active bool) string {
	if len(items) == 0 {
		return "–"
	}
	var rows []string
	for i, item := range items {
		if active && i == selected {
			rows = append(rows, selectedStyle.Render("> "+item))
		} else {
			rows = append(rows, "  "+item)
		}
	}
	return strings.Join(rows, "\n")
}

// renderAliasBody renders aliases and action toggles.
func (m model) renderAliasBody(active bool) string {
	if len(m.aliases) == 0 {
		return "–"
	}

	var rows []string
	for i, alias := range m.aliases {
		var row strings.Builder
		if active && i == m.aliasIndex {
			row.WriteString(selectedStyle.Render("> " + alias))
			row.WriteString("  ")
			for j, action := range aliasActions {
				if m.actionFocus && j == m.actionIndex {
					row.WriteString(selectedStyle.Render(action))
				} else {
					row.WriteString(action)
				}
				if j < len(aliasActions)-1 {
					row.WriteString("  ")
				}
			}
		} else {
			row.WriteString("  " + alias)
		}
		rows = append(rows, row.String())
	}
	return strings.Join(rows, "\n")
}
