package tui

import "strings"

// logLineMsg carries log lines into the Bubble Tea update loop.
type logLineMsg struct {
	line string
}

// logForwarder redirects log output into Bubble Tea messages.
type logForwarder struct {
	ch chan<- string
}

// Write implements io.Writer to forward logs into the program loop.
func (w logForwarder) Write(p []byte) (int, error) {
	text := strings.TrimSpace(string(p))
	if text == "" {
		return len(p), nil
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		select {
		case w.ch <- line:
		default:
			// Drop if channel is full to avoid blocking.
		}
	}
	return len(p), nil
}
