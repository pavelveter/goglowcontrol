package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// executeCommand executes a command for a device
//
// Parameters:
//   - ip: device IP address
//   - action: action to execute
//   - param: action parameter
func executeCommand(ip string, action string, param string) {
	target, base := parseTargetAndAction(action)

	switch base {
	case "on":
		sendByTarget(ip, target, "set_power", []interface{}{"on", "smooth", defaultSmoothDuration})
	case "off":
		sendByTarget(ip, target, "set_power", []interface{}{"off", "smooth", defaultSmoothDuration})
	case "color":
		if colorInt, ok := colorToInt(param); ok {
			sendByTarget(ip, target, "set_scene", []interface{}{"color", colorInt, maxBrightness})
		} else {
			log.Printf("[%s] Unknown color: %s", ip, param)
		}
	case "t":
		if num, err := strconv.Atoi(param); err == nil {
			if num < minTemperature || num > maxTemperature {
				log.Printf("[%s] Color temperature must be %d..%d", ip, minTemperature, maxTemperature)
				return
			}
			sendByTarget(ip, target, "set_ct_abx", []interface{}{num, "smooth", defaultSmoothDuration})
		} else {
			log.Printf("[%s] Invalid temperature value: %s", ip, param)
		}
	case "disco":
		sendByTarget(ip, target, "start_cf", []interface{}{50, 0, "300, 1, 255, 100, 100, 1, 32768, 100, 100, 1, 16711680, 100"})
	case "sunrise":
		sendByTarget(ip, target, "start_cf", []interface{}{3, 1, "50, 1, 16731392, 1, 360000, 2, 1700, 10, 540000, 2, 2700, 100"})
	case "dim":
		sendByTarget(ip, target, "set_bright", []interface{}{defaultDimValue})
	case "undim":
		sendByTarget(ip, target, "set_bright", []interface{}{defaultBrightValue})
	case "brightness":
		if num, err := strconv.Atoi(param); err == nil {
			if num < minBrightness || num > maxBrightness {
				log.Printf("[%s] Brightness must be %d..%d", ip, minBrightness, maxBrightness)
				return
			}
			sendByTarget(ip, target, "set_bright", []interface{}{num})
		} else {
			log.Printf("[%s] Invalid brightness value: %s", ip, param)
		}
	case "scene":
		executeScene(param)
	default:
		if num, err := strconv.Atoi(base); err == nil {
			if num >= minBrightness && num <= maxBrightness {
				sendByTarget(ip, target, "set_bright", []interface{}{num})
			} else if num >= minTemperature && num <= maxTemperature {
				sendByTarget(ip, target, "set_ct_abx", []interface{}{num, "smooth", defaultSmoothDuration})
			} else {
				log.Printf("[%s] Invalid numeric value: %s (must be brightness %d-%d or temperature %d-%d)",
					ip, base, minBrightness, maxBrightness, minTemperature, maxTemperature)
			}
			return
		}
		if strings.HasPrefix(base, "notify-") {
			colorName := strings.TrimPrefix(base, "notify-")
			if colorInt, ok := colorToInt(colorName); ok {
				sendByTarget(ip, target, "start_cf", []interface{}{5, 0,
					fmt.Sprintf("100, 1, %d, 100, 100, 1, %d, 1", colorInt, colorInt)})
			} else {
				log.Printf("[%s] Unknown color for notification: %s", ip, colorName)
			}
			return
		}
		if colorInt, ok := colorToInt(base); ok {
			sendByTarget(ip, target, "set_scene", []interface{}{"color", colorInt, maxBrightness})
			return
		}
		log.Printf("[%s] Unknown action: %s", ip, action)
	}
}

// executeScene executes a scene
//
// Parameters:
//   - sceneName: name of the scene to execute
func executeScene(sceneName string) {
	scene, ok := scenes[sceneName]
	if !ok {
		log.Printf("Scene '%s' not found", sceneName)
		return
	}
	for _, command := range scene.Commands {
		parts := strings.Fields(command)
		if len(parts) < 2 {
			continue
		}
		ip := parts[0]
		action := parts[1]
		param := ""
		if len(parts) > 2 {
			param = parts[2]
		}

		var ips []string
		if strings.HasPrefix(ip, "@") {
			if groupIPs, ok := bulbGroups[ip]; ok {
				ips = strings.Fields(groupIPs)
			} else {
				log.Printf("Alias '%s' not found in scene", ip)
				continue
			}
		} else {
			ips = []string{ip}
		}

		for _, resolvedIP := range ips {
			var ipList []string
			addArgToIPs(resolvedIP, &ipList, nil, nil)
			for _, finalIP := range ipList {
				executeCommand(finalIP, action, param)
			}
		}
	}
}
