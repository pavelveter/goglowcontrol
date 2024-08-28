package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Scene struct {
	Name     string
	Commands []string
}

var colors string
var bulbs string
var scenes map[string]Scene
var bulbGroups map[string]string
var execDir string

func init() {
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

    colors, err = readFileFromDirs("colors.txt", execDir, currentDir)
    if err != nil {
        log.Printf("Error reading colors.txt: %v", err)
        os.Exit(1)
    }

    bulbs, err = readFileFromDirs("bulbs.txt", execDir, currentDir)
    if err != nil {
        log.Printf("Error reading bulbs.txt: %v", err)
        os.Exit(1)
    }

    bulbGroups = loadBulbGroups()

    scenes = loadScenesFromDirs("scenes.txt", execDir, currentDir)
}

func readFileFromDirs(filename string, dirs ...string) (string, error) {
    for _, dir := range dirs {
        path := filepath.Join(dir, filename)
        data, err := os.ReadFile(path)
        if err == nil {
            return strings.TrimSpace(string(data)), nil
        }
    }
    return "", fmt.Errorf("file %s not found", filename)
}

func loadScenesFromDirs(filename string, dirs ...string) map[string]Scene {
    for _, dir := range dirs {
        path := filepath.Join(dir, filename)
        content, err := os.ReadFile(path)
        if err == nil {
            return parseScenes(string(content))
        }
    }
    log.Printf("Scenes file %s not found, using empty scenes map", filename)
    return make(map[string]Scene)
}

func parseScenes(content string) map[string]Scene {
    scenes := make(map[string]Scene)
    lines := strings.Split(content, "\n")
    for _, line := range lines {
        parts := strings.SplitN(line, ":", 2)
        if len(parts) != 2 {
            continue
        }
        name := strings.TrimSpace(parts[0])
        commands := strings.Split(parts[1], ",")
        for i := range commands {
            commands[i] = strings.TrimSpace(commands[i])
        }
        scenes[name] = Scene{Name: name, Commands: commands}
    }
    return scenes
}

func getLocalNetworkPrefix() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return strings.Join(strings.Split(ipnet.IP.String(), ".")[:3], ".")
			}
		}
	}
	return ""
}

func printHelp() {
	net := getLocalNetworkPrefix()
	scriptName := filepath.Base(os.Args[0])

	var colorList, aliasList []string
	for _, line := range strings.Split(strings.TrimSpace(colors), "\n") {
		colorList = append(colorList, strings.Split(line, ";")[0])
	}
	for _, line := range strings.Split(strings.TrimSpace(bulbs), "\n") {
		aliasList = append(aliasList, strings.Split(line, ";")[0])
	}
	var sceneList []string
	for sceneName := range scenes {
		sceneList = append(sceneList, sceneName)
	}

	helpText := fmt.Sprintf(`Usage: %s <ip|@alias> <command> -- utility to control Yeelight smart bulb(s) over Wi-Fi

'ip' can be a single value, several values, or ranges of IP addresses,
'@alias' can be an alias of a bulb or a group of bulbs,
'command' can have one of the following values:

on - turn on the light
off - turn off the light
[color] <color> - set the color to <color>, key is optional
[t] <number> - set the white light temperature to 1700..6500, key is optional
disco - turn on disco mode
sunrise - turn on sunrise mode
notify-<color> - notification in <color>
dim - dim light to brightness 5
undim - reset light to brightness 100
scene <name> - execute scene
[brightness] <level> - from 1 (dimmest) to 100 (brightest), key is optional

Examples: %s %s.1 on -- turn on the single bulb
          %s %s.1-2 %s.4 color red -- give three bulbs the color red
          %s %s.1 %s.3 50 -- set the brightness of two bulbs to 50%%
          %s %s.2 4100 -- set the bulb's white temperature to 4100
          %s @room notify-blue -- notify via the room bulbs with blue color
          %s scene evening -- execute the evening scene`, scriptName, scriptName, net, scriptName, net, net, scriptName, net, net, scriptName, net, scriptName, scriptName)

	fmt.Println(helpText)

	printWrappedList("Colors:", colorList, 80)
	printWrappedList("Aliases:", aliasList, 80)
	printWrappedList("Scenes:", sceneList, 80)

	os.Exit(0)
}

func printWrappedList(title string, items []string, maxWidth int) {
	fmt.Println(title)
	line := "  "
	for _, item := range items {
		if len(line)+len(item)+2 > maxWidth {
			fmt.Println(line)
			line = "  "
		}
		line += item + ", "
	}
	if line != "  " {
		fmt.Println(strings.TrimSuffix(line, ", "))
	}
}

func addArgToIPs(arg string, ips *[]string, command *string, param *string) {
	if match, _ := regexp.MatchString(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]+$`, arg); match {
		parts := strings.Split(arg, "-")
		ip := strings.Split(parts[0], ".")
		start, _ := strconv.Atoi(ip[3])
		end, _ := strconv.Atoi(parts[1])
		for i := start; i <= end; i++ {
			*ips = append(*ips, fmt.Sprintf("%s.%s.%s.%d", ip[0], ip[1], ip[2], i))
		}
	} else if match, _ := regexp.MatchString(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`, arg); match {
		*ips = append(*ips, arg)
	} else {
		if *command == "" {
			*command = arg
		} else {
			*param = arg
		}
	}
}

func sendCommand(ip, method string, params []interface{}) {
	message := map[string]interface{}{
		"id":     1,
		"method": method,
		"params": params,
	}
	jsonMessage, _ := json.Marshal(message)
	jsonMessage = append(jsonMessage, '\r', '\n')

	conn, err := net.DialTimeout("tcp", ip+":55443", time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.Write(jsonMessage)
	if err != nil {
		log.Printf("Error sending command to %s: %v", ip, err)
	}
}

func colorToInt(colorName string) (int64, bool) {
	for _, line := range strings.Split(strings.TrimSpace(colors), "\n") {
		parts := strings.Split(line, ";")
		if strings.EqualFold(parts[0], colorName) {
			value, err := strconv.ParseInt(parts[1][1:], 16, 32)
			if err != nil {
				return 0, false
			}
			return value, true
		}
	}
	return 0, false
}

func executeCommand(ip string, action string, param string) {
	switch action {
	case "on":
		sendCommand(ip, "set_power", []interface{}{"on", "smooth", 500})
	case "off":
		sendCommand(ip, "set_power", []interface{}{"off", "smooth", 500})
	case "color":
		if colorInt, ok := colorToInt(param); ok {
			sendCommand(ip, "set_scene", []interface{}{"color", colorInt, 100})
		} else {
			log.Printf("Unknown color: %s", param)
		}
	case "t":
		if num, err := strconv.Atoi(param); err == nil && num >= 1700 && num <= 6500 {
			sendCommand(ip, "set_ct_abx", []interface{}{num, "smooth", 500})
		} else {
			log.Println("Color temperature must be between 1700 and 6500")
		}
	case "disco":
		sendCommand(ip, "start_cf", []interface{}{50, 0, "300, 1, 255, 100, 100, 1, 32768, 100, 100, 1, 16711680, 100"})
	case "sunrise":
		sendCommand(ip, "start_cf", []interface{}{3, 1, "50, 1, 16731392, 1, 360000, 2, 1700, 10, 540000, 2, 2700, 100"})
	case "dim":
		sendCommand(ip, "set_bright", []interface{}{5})
	case "undim":
		sendCommand(ip, "set_bright", []interface{}{100})
	case "brightness":
		if num, err := strconv.Atoi(param); err == nil && num >= 1 && num <= 100 {
			sendCommand(ip, "set_bright", []interface{}{num})
		} else {
			log.Println("Brightness must be between 1 and 100")
			printHelp()
		}
	case "scene":
		executeScene(param)
	default:
		if num, err := strconv.Atoi(action); err == nil && num >= 1700 && num <= 6500 {
			sendCommand(ip, "set_ct_abx", []interface{}{num, "smooth", 500})
		} else if strings.HasPrefix(action, "notify-") {
			if colorInt, ok := colorToInt(action[7:]); ok {
				sendCommand(ip, "start_cf", []interface{}{5, 0, fmt.Sprintf("100, 1, %d, 100, 100, 1, %d, 1", colorInt, colorInt)})
			} else {
				log.Printf("Unknown notification color: %s", action[7:])
			}
		} else if colorInt, ok := colorToInt(action); ok {
			sendCommand(ip, "set_scene", []interface{}{"color", colorInt, 100})
		} else {
			log.Printf("Unknown action: %s", action)
		}
	}
}

func executeScene(sceneName string) {
	scene, ok := scenes[sceneName]
	if !ok {
		log.Printf("Scene '%s' not found", sceneName)
		return
	}
	for _, command := range scene.Commands {
		parts := strings.Fields(command)
		if len(parts) < 2 {
			log.Printf("Skipping invalid command: %s", command)
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
				log.Printf("Unknown alias: %s", ip)
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

func loadBulbGroups() map[string]string {
    bulbGroups := make(map[string]string)
    tempGroups := make(map[string][]string)

    content, err := os.ReadFile(filepath.Join(execDir, "bulbs.txt"))
    if err != nil {
        log.Printf("Error reading bulbs.txt from execDir: %v", err)
        currentDir, _ := os.Getwd()
        content, err = os.ReadFile(filepath.Join(currentDir, "bulbs.txt"))
        if err != nil {
            log.Printf("Error reading bulbs.txt from current directory: %v", err)
            return bulbGroups
        }
    }

    scanner := bufio.NewScanner(bytes.NewReader(content))
    for scanner.Scan() {
        parts := strings.SplitN(scanner.Text(), ":", 2)
        if len(parts) == 2 {
            alias := strings.TrimSpace(parts[0])
            ips := strings.Fields(parts[1])
            tempGroups[alias] = ips
        }
    }

    // Resolve aliases
    for alias, ips := range tempGroups {
        resolvedIPs := resolveAliases(ips, tempGroups)
        bulbGroups[alias] = strings.Join(resolvedIPs, " ")
    }

    return bulbGroups
}

func resolveAliases(ips []string, groups map[string][]string) []string {
    var resolvedIPs []string
    for _, ip := range ips {
        if strings.HasPrefix(ip, "@") {
            if nestedIPs, ok := groups[ip]; ok {
                resolvedIPs = append(resolvedIPs, resolveAliases(nestedIPs, groups)...)
            } else {
                resolvedIPs = append(resolvedIPs, ip) // Unknown alias, leave as is
            }
        } else {
            resolvedIPs = append(resolvedIPs, ip)
        }
    }
    return resolvedIPs
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	// Special case for scene command
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

	// If command is a known color name, assume command 'color'
	for _, line := range strings.Split(strings.TrimSpace(colors), "\n") {
		if strings.EqualFold(strings.Split(line, ";")[0], command) {
			param = command
			command = "color"
			break
		}
	}

	// If command is a number, assume commands 't' or 'brightness'
	if num, err := strconv.Atoi(command); err == nil {
		if num >= 1 && num <= 100 {
			param = command
			command = "brightness"
		} else {
			param = command
			command = "t"
		}
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