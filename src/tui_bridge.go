package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"glowcontrol/src/tui"
)

// runTUI constructs dependencies and launches the TUI.
func runTUI() error {
	scenesPath := filepath.Join("cfg", "scenes.txt")
	deps := tui.Deps{
		Colors:         collectColorNames(),
		Aliases:        collectAliases(),
		Scenes:         collectSceneNames(),
		ColorHex:       colorMap.get,
		HexToRGB:       hexToRGB,
		ColorReset:     colorReset,
		MinBrightness:  minBrightness,
		MaxBrightness:  maxBrightness,
		MinTemperature: minTemperature,
		MaxTemperature: maxTemperature,
		ResolveTargets: resolveTargets,
		Execute:        executeCommand,
		SceneCommands:  collectSceneCommandsMap(),
		State:          tui.NewState(minBrightness, maxBrightness, minTemperature, maxTemperature),
		SaveScene: func(name string, commands []string) (bool, error) {
			return saveSceneToFile(scenesPath, name, commands)
		},
		RunScene: func(name string) (string, error) {
			executeScene(name)
			return fmt.Sprintf("Scene %s executed", name), nil
		},
	}
	return tui.Run(deps)
}

func saveSceneToFile(path, name string, commands []string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("scene name cannot be empty")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("creating cfg directory: %w", err)
	}

	entry := fmt.Sprintf("%s:", name)
	if len(commands) > 0 {
		entry = fmt.Sprintf("%s: %s", name, strings.Join(commands, ", "))
	}
	updated := false

	var lines []string
	if data, err := os.ReadFile(path); err == nil {
		lines = strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("reading scenes file: %w", err)
	}

	out := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && line == "" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) > 0 && strings.TrimSpace(parts[0]) == name {
			if !updated {
				out = append(out, entry)
				updated = true
			}
			continue
		}
		out = append(out, line)
	}

	if !updated {
		out = append(out, entry)
	}

	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return updated, fmt.Errorf("writing scenes file: %w", err)
	}

	if scenes == nil {
		scenes = make(map[string]Scene)
	}
	scenes[name] = Scene{Name: name, Commands: commands}
	return updated, nil
}

// collectColorNames builds a sorted list of available colors.
func collectColorNames() []string {
	colorMap.mu.RLock()
	defer colorMap.mu.RUnlock()

	colors := make([]string, 0, len(colorMap.colors))
	for name := range colorMap.colors {
		colors = append(colors, name)
	}
	sort.Strings(colors)
	return colors
}

// collectAliases returns aliases ordered by multi-IP first, then size, then name.
func collectAliases() []string {
	type meta struct {
		name  string
		count int
	}
	var metas []meta
	for alias, ips := range bulbGroups {
		metas = append(metas, meta{name: alias, count: len(strings.Fields(ips))})
	}

	sort.SliceStable(metas, func(i, j int) bool {
		mi, mj := metas[i], metas[j]
		if (mi.count > 1) != (mj.count > 1) {
			return mi.count > 1
		}
		if mi.count != mj.count {
			return mi.count > mj.count
		}
		return mi.name < mj.name
	})

	result := make([]string, 0, len(metas))
	for _, m := range metas {
		result = append(result, m.name)
	}
	return result
}

// collectSceneNames returns a sorted list of scenes.
func collectSceneNames() []string {
	var names []string
	for name := range scenes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// collectSceneCommandsMap copies scene commands for TUI state tracking.
func collectSceneCommandsMap() map[string][]string {
	result := make(map[string][]string, len(scenes))
	for name, scene := range scenes {
		cmds := make([]string, len(scene.Commands))
		copy(cmds, scene.Commands)
		result[name] = cmds
	}
	return result
}

// resolveTargets expands aliases and ranges into concrete IPs.
func resolveTargets(targets []string) ([]string, error) {
	var ips []string
	seen := make(map[string]bool)

	for _, raw := range targets {
		candidates := []string{raw}
		if groupIPs, ok := bulbGroups[raw]; ok {
			candidates = strings.Fields(groupIPs)
		}

		for _, candidate := range candidates {
			switch {
			case isValidIP(candidate):
				if !seen[candidate] {
					ips = append(ips, candidate)
					seen[candidate] = true
				}
			case parseIPRange(candidate) != nil:
				for _, ip := range parseIPRange(candidate) {
					if !seen[ip] {
						ips = append(ips, ip)
						seen[ip] = true
					}
				}
			default:
				return nil, fmt.Errorf("invalid target: %s", candidate)
			}
		}
	}
	return ips, nil
}
