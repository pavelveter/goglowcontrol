package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var useColors = supportsColor()
var exitFunc = os.Exit

// supportsColor checks if the terminal supports colors
func supportsColor() bool {
	// Check NO_COLOR env
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Ensure stdout is a TTY
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// Pipes or files likely do not support color
	if (info.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	// Check TERM
	term := os.Getenv("TERM")
	if term == "" {
		return false
	}
	// Supported terminals (including true color capable)
	supportedTerms := []string{"xterm", "screen", "tmux", "vt100", "color", "ansi", "linux",
		"xterm-256color", "xterm-256", "256color", "truecolor", "ghostty", "iterm", "termux"}
	for _, t := range supportedTerms {
		if strings.Contains(strings.ToLower(term), t) {
			return true
		}
	}
	// Check COLORTERM for true color support
	if colorterm := os.Getenv("COLORTERM"); colorterm != "" {
		if strings.Contains(strings.ToLower(colorterm), "truecolor") ||
			strings.Contains(strings.ToLower(colorterm), "24bit") {
			return true
		}
	}
	return false
}

// colorize applies ANSI color to text if supported
func colorize(text, color string) string {
	if !useColors {
		return text
	}
	return color + text + colorReset
}

// hexToRGB converts hex color to RGB ANSI escape code for true color (24-bit)
func hexToRGB(hex int64) string {
	if !useColors {
		return ""
	}
	// Extract R, G, B from int64
	r := int((hex >> 16) & 0xFF)
	g := int((hex >> 8) & 0xFF)
	b := int(hex & 0xFF)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// getLocalNetworkPrefix returns the prefix of the local network
func getLocalNetworkPrefix() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipParts := strings.Split(ipnet.IP.String(), ".")
				if len(ipParts) >= 3 {
					return strings.Join(ipParts[:3], ".")
				}
			}
		}
	}
	return ""
}

// printHelp prints usage information
func printHelp() {
	netp := getLocalNetworkPrefix()
	scriptName := filepath.Base(os.Args[0])

	var colorList, aliasList []string

	// Use color cache to build the list
	colorMap.mu.RLock()
	for colorName := range colorMap.colors {
		colorList = append(colorList, colorName)
	}
	colorMap.mu.RUnlock()

	for _, line := range strings.Split(strings.TrimSpace(bulbsData), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 0 {
			aliasList = append(aliasList, strings.TrimSpace(parts[0]))
		}
	}

	var sceneList []string
	for sceneName := range scenes {
		sceneList = append(sceneList, sceneName)
	}

	fmt.Print(colorize("Usage: ", colorBold+colorCyan))
	fmt.Printf("%s%s <ip|@alias|range> <command> [param]\n\n",
		colorize(scriptName, colorBold), colorReset)

	fmt.Println(colorize("Commands:", colorBold))
	fmt.Printf("  %s: %s, %s, %s <c>, %s <1700..6500>, %s <1..100>, %s, %s, %s, %s, %s\n",
		colorize("Main (default)", colorYellow),
		colorize("on", colorGreen), colorize("off", colorGreen),
		colorize("color", colorCyan), colorize("t", colorCyan),
		colorize("brightness", colorCyan), colorize("dim", colorGreen),
		colorize("undim", colorGreen), colorize("disco", colorGreen),
		colorize("sunrise", colorGreen), colorize("notify-<c>", colorPurple))
	fmt.Printf("  %s:     %s, %s, %s <c>, %s <1700..6500>, %s <1..100>, %s, %s, %s, %s, %s\n",
		colorize("BG channel", colorYellow),
		colorize("bg-on", colorGreen), colorize("bg-off", colorGreen),
		colorize("bg-color", colorCyan), colorize("bg-t", colorCyan),
		colorize("bg-brightness", colorCyan), colorize("bg-dim", colorGreen),
		colorize("bg-undim", colorGreen), colorize("bg-disco", colorGreen),
		colorize("bg-sunrise", colorGreen), colorize("bg-notify-<c>", colorPurple))
	fmt.Printf("  %s:  %s, %s, %s <1..100>, %s, %s\n",
		colorize("Both channels", colorYellow),
		colorize("both-on", colorGreen), colorize("both-off", colorGreen),
		colorize("both-brightness", colorCyan), colorize("both-dim", colorGreen),
		colorize("both-undim", colorGreen))
	fmt.Printf("  %s:         %s <name>\n\n",
		colorize("Scenes", colorYellow), colorize("scene", colorCyan))

	fmt.Println(colorize("Examples:", colorBold))
	fmt.Printf("  %s %s %s\n", scriptName, colorize(netp+".50", colorBlue), colorize("on", colorGreen))
	fmt.Printf("  %s %s %s\n", scriptName, colorize(netp+".50", colorBlue), colorize("bg-on", colorGreen))
	fmt.Printf("  %s %s %s %s\n", scriptName, colorize(netp+".51-53", colorBlue), colorize("color", colorCyan), colorize("red", colorRed))
	fmt.Printf("  %s %s %s %s\n", scriptName, colorize("@room", colorBlue), colorize("both-brightness", colorCyan), colorize("40", colorYellow))
	fmt.Printf("  %s %s %s\n\n", scriptName, colorize("scene", colorCyan), colorize("evening", colorPurple))

	printWrappedListColorized(colorize("Colors:", colorBold), colorList, 80)
	printWrappedList(colorize("Aliases:", colorBold), aliasList, 80)
	printWrappedList(colorize("Scenes:", colorBold), sceneList, 80)
	exitFunc(0)
}

// printWrappedList prints items with wrapping
func printWrappedList(title string, items []string, maxWidth int) {
	fmt.Println(title)
	line := "  "
	for i, item := range items {
		// Alternate colors for list items
		var coloredItem string
		if useColors {
			var itemColor string
			switch i % 4 {
			case 0:
				itemColor = colorCyan
			case 1:
				itemColor = colorGreen
			case 2:
				itemColor = colorYellow
			case 3:
				itemColor = colorPurple
			}
			coloredItem = itemColor + item + colorReset
		} else {
			coloredItem = item
		}
		// Width calculation uses plain text length (no ANSI codes)
		itemLen := len(item)
		// Calculate plain line length without ANSI escape codes
		plainLine := ansiEscapeRegex.ReplaceAllString(line, "")
		if len(plainLine)+itemLen+2 > maxWidth {
			fmt.Println(strings.TrimSuffix(line, ", "))
			line = "  "
		}
		line += coloredItem + ", "
	}
	if strings.TrimSpace(line) != "" && line != "  " {
		fmt.Println(strings.TrimSuffix(line, ", "))
	}
}

// printWrappedListColorized prints colors using their actual values
func printWrappedListColorized(title string, items []string, maxWidth int) {
	fmt.Println(title)
	line := "  "
	for _, item := range items {
		// Get actual color from cache
		var coloredItem string
		if useColors {
			if hexColor, ok := colorMap.get(item); ok {
				// Use the real color from hex
				coloredItem = hexToRGB(hexColor) + item + colorReset
			} else {
				// Fallback to default color
				coloredItem = colorCyan + item + colorReset
			}
		} else {
			coloredItem = item
		}
		// Width calculation uses plain text length (no ANSI codes)
		itemLen := len(item)
		// Calculate plain line length without ANSI escape codes
		plainLine := ansiEscapeRegex.ReplaceAllString(line, "")
		if len(plainLine)+itemLen+2 > maxWidth {
			fmt.Println(strings.TrimSuffix(line, ", "))
			line = "  "
		}
		line += coloredItem + ", "
	}
	if strings.TrimSpace(line) != "" && line != "  " {
		fmt.Println(strings.TrimSuffix(line, ", "))
	}
}
