package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	// colorsData data from colors.txt file
	colorsData string
	// bulbsData data from bulbs.txt file
	bulbsData string
	// scenes map of scenes
	scenes map[string]Scene
	// bulbGroups map of bulb groups
	bulbGroups map[string]string
	// execDir executable file directory
	execDir string
)

// initConfig initializes the application configuration
func initConfig() {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable path: %v", err)
	}
	execDir = filepath.Dir(execPath)
	currentDir, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting current directory: %v", err)
		os.Exit(1)
	}
	parentDir := filepath.Dir(currentDir)
	grandParentDir := filepath.Dir(parentDir)

	colorsData, err = readFileFromDirs("colors.txt", execDir, currentDir, parentDir, grandParentDir)
	if err != nil {
		log.Printf("Error reading colors.txt: %v", err)
		os.Exit(1)
	}

	bulbsData, err = readFileFromDirs("bulbs.txt", execDir, currentDir, parentDir, grandParentDir)
	if err != nil {
		log.Printf("Error reading bulbs.txt: %v", err)
		os.Exit(1)
	}

	// Load color cache
	if err := colorMap.load(colorsData); err != nil {
		log.Printf("Warning: error loading color cache: %v", err)
	}

	bulbGroups = loadBulbGroups()
	scenes = loadScenesFromDirs("scenes.txt", execDir, currentDir, parentDir, grandParentDir)
}

// readFileFromDirs reads a file from specified directories (preferring cfg subdir)
//
// Parameters:
//   - filename: name of the file to read
//   - dirs: list of directories to search for the file
//
// Returns:
//   - string: file content
//   - error: error during file reading
func readFileFromDirs(filename string, dirs ...string) (string, error) {
	for _, dir := range dirs {
		for _, path := range []string{
			filepath.Join(dir, "cfg", filename),
			filepath.Join(dir, filename),
		} {
			data, err := os.ReadFile(path)
			if err == nil {
				return strings.TrimSpace(string(data)), nil
			}
		}
	}
	return "", fmt.Errorf("file %s not found", filename)
}

// loadScenesFromDirs loads scenes from file in specified directories
//
// Parameters:
//   - filename: name of the scenes file
//   - dirs: list of directories to search for the file
//
// Returns:
//   - map[string]Scene: map of scenes
func loadScenesFromDirs(filename string, dirs ...string) map[string]Scene {
	for _, dir := range dirs {
		for _, path := range []string{
			filepath.Join(dir, "cfg", filename),
			filepath.Join(dir, filename),
		} {
			content, err := os.ReadFile(path)
			if err == nil {
				return parseScenes(string(content))
			}
		}
	}
	log.Printf("Scenes file %s not found, using empty scenes map", filename)
	return make(map[string]Scene)
}

// parseScenes parses scenes file content
//
// Parameters:
//   - content: scenes file content
//
// Returns:
//   - map[string]Scene: map of scenes
func parseScenes(content string) map[string]Scene {
	result := make(map[string]Scene)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		cmds := strings.Split(parts[1], ",")
		for i := range cmds {
			cmds[i] = strings.TrimSpace(cmds[i])
		}
		result[name] = Scene{Name: name, Commands: cmds}
	}
	return result
}

// loadBulbGroups loads bulb groups from bulbs.txt file
//
// Returns:
//   - map[string]string: map of bulb groups
func loadBulbGroups() map[string]string {
	bulbGroups := make(map[string]string)
	tempGroups := make(map[string][]string)

	currentDir, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: could not get working directory: %v", err)
		return bulbGroups
	}
	parentDir := filepath.Dir(currentDir)
	grandParentDir := filepath.Dir(parentDir)

	content, err := readFileFromDirs("bulbs.txt", execDir, currentDir, parentDir, grandParentDir)
	if err != nil {
		log.Printf("Warning: could not read bulbs.txt: %v", err)
		return bulbGroups
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			alias := strings.TrimSpace(parts[0])
			ips := strings.Fields(parts[1])
			tempGroups[alias] = ips
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Warning: error reading bulbs.txt: %v", err)
	}

	for alias, ips := range tempGroups {
		resolvedIPs := resolveAliases(ips, tempGroups)
		bulbGroups[alias] = strings.Join(resolvedIPs, " ")
	}
	return bulbGroups
}

// resolveAliases resolves aliases in IP address list
//
// Parameters:
//   - ips: list of IP addresses and aliases
//   - groups: map of groups
//
// Returns:
//   - []string: list of resolved IP addresses
func resolveAliases(ips []string, groups map[string][]string) []string {
	var resolvedIPs []string
	seen := make(map[string]bool)
	var expand func(items []string)
	expand = func(items []string) {
		for _, ip := range items {
			if strings.HasPrefix(ip, "@") {
				if nested, ok := groups[ip]; ok {
					expand(nested)
				}
			} else {
				if !seen[ip] {
					seen[ip] = true
					resolvedIPs = append(resolvedIPs, ip)
				}
			}
		}
	}
	expand(ips)
	return resolvedIPs
}
