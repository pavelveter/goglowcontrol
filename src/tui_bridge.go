package main

import (
	"fmt"
	"sort"
	"strings"

	"glowcontrol/src/tui"
)

// runTUI constructs dependencies and launches the TUI.
func runTUI() error {
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
		RunScene: func(name string) (string, error) {
			executeScene(name)
			return fmt.Sprintf("Scene %s executed", name), nil
		},
	}
	return tui.Run(deps)
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
