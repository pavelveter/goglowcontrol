package tui

import "strings"

type logLineMsg struct {
	line string
}

type logForwarder struct {
	ch chan<- string
}

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
