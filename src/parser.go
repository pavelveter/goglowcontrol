package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Regular expressions for parsing IP addresses
var (
	ipRangeRegex    = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)\.(\d+)-(\d+)$`)
	ipRegex         = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)\.(\d+)$`)
	ansiEscapeRegex = regexp.MustCompile(`\033\[[0-9;]+m`)
)

// parseIPRange parses IP ranges like "192.168.1.1-10"
func parseIPRange(ipRange string) []string {
	matches := ipRangeRegex.FindStringSubmatch(ipRange)
	if len(matches) != 6 {
		return nil
	}
	base := matches[1] + "." + matches[2] + "." + matches[3]
	start, err1 := strconv.Atoi(matches[4])
	end, err2 := strconv.Atoi(matches[5])
	if err1 != nil || err2 != nil || start > end {
		return nil
	}

	var ips []string
	for i := start; i <= end; i++ {
		ips = append(ips, fmt.Sprintf("%s.%d", base, i))
	}
	return ips
}

// isValidIP checks whether a string is a valid IP address
func isValidIP(ip string) bool {
	return ipRegex.MatchString(ip)
}

// addArgToIPs adds IP addresses from an argument or determines command/param
func addArgToIPs(arg string, ips *[]string, command *string, param *string) {
	if parsedIPs := parseIPRange(arg); parsedIPs != nil {
		*ips = append(*ips, parsedIPs...)
		return
	}
	if isValidIP(arg) {
		*ips = append(*ips, arg)
		return
	}
	// Not an IP, so treat as command or parameter
	if command != nil && *command == "" {
		*command = arg
	} else if param != nil && *param == "" {
		*param = arg
	}
}

// parseTargetAndAction parses an action and determines the target channel
func parseTargetAndAction(action string) (Target, string) {
	a := strings.ToLower(strings.TrimSpace(action))
	switch {
	case strings.HasPrefix(a, "bg-") || strings.HasPrefix(a, "bg_"):
		base := strings.TrimPrefix(a, "bg-")
		base = strings.TrimPrefix(base, "bg_")
		return BG, base
	case strings.HasPrefix(a, "both-") || strings.HasPrefix(a, "both_"):
		base := strings.TrimPrefix(a, "both-")
		base = strings.TrimPrefix(base, "both_")
		return Both, base
	default:
		return Main, a
	}
}
