package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

func init() {
	initConfig()
}

// main entry point of the application
func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	if os.Args[1] == "tui" {
		if err := runTUI(); err != nil {
			log.Fatalf("TUI failed: %v", err)
		}
		return
	}

	if os.Args[1] == "scene" && len(os.Args) == 3 {
		executeScene(os.Args[2])
		return
	}

	var ips []string
	var command, param string

	for _, arg := range os.Args[1:] {
		if groupIPs, ok := bulbGroups[arg]; ok {
			for _, groupIP := range strings.Fields(groupIPs) {
				addArgToIPs(groupIP, &ips, &command, &param)
			}
		} else {
			addArgToIPs(arg, &ips, &command, &param)
		}
	}

	if command == "" || command == "help" || command == "--help" || command == "-h" {
		printHelp()
		return
	}

	// Check if command is a color name
	if _, ok := colorMap.get(command); ok {
		param = command
		command = "color"
	} else if num, err := strconv.Atoi(command); err == nil {
		// If command is a number: brightness (1..100) or t (1700..6500)
		if num >= minBrightness && num <= maxBrightness {
			param = command
			command = "brightness"
		} else if num >= minTemperature && num <= maxTemperature {
			param = command
			command = "t"
		} else {
			log.Printf("Invalid numeric command: %s (must be brightness %d-%d or temperature %d-%d)",
				command, minBrightness, maxBrightness, minTemperature, maxTemperature)
			return
		}
	}

	if len(ips) == 0 {
		log.Println("No valid IP addresses found")
		return
	}

	var wg sync.WaitGroup
	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			executeCommand(ip, command, param)
		}(ip)
	}
	wg.Wait()
}
