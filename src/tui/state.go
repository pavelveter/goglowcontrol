package tui

import (
	"strconv"
	"strings"
	"sync"
)

// LightState keeps the last known parameters applied to a target.
type LightState struct {
	Power       string
	Color       string
	Brightness  int
	Temperature int
}

// State tracks light states in a concurrency-safe map.
type State struct {
	mu             sync.Mutex
	lights         map[string]LightState
	minBrightness  int
	maxBrightness  int
	minTemperature int
	maxTemperature int
}

// NewState creates a State with range information for numeric parsing.
func NewState(minBrightness, maxBrightness, minTemperature, maxTemperature int) *State {
	return &State{
		lights:         make(map[string]LightState),
		minBrightness:  minBrightness,
		maxBrightness:  maxBrightness,
		minTemperature: minTemperature,
		maxTemperature: maxTemperature,
	}
}

// Snapshot returns a copy of the stored light states.
func (s *State) Snapshot() map[string]LightState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]LightState, len(s.lights))
	for alias, state := range s.lights {
		out[alias] = state
	}
	return out
}

// RecordAction saves on/off style commands.
func (s *State) RecordAction(alias, action string) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action != "on" && action != "off" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(alias)
	state.Power = action
	s.lights[alias] = state
}

// RecordCommand saves parameterized commands like color/brightness/temperature.
func (s *State) RecordCommand(alias, command, param string) {
	s.apply(alias, strings.ToLower(strings.TrimSpace(command)), strings.TrimSpace(param))
}

// RecordScene processes a list of scene commands and saves resulting state.
func (s *State) RecordScene(commands []string) {
	if len(commands) == 0 {
		return
	}
	for _, raw := range commands {
		alias, action, param := splitSceneCommand(raw)
		if alias == "" || action == "" {
			continue
		}
		s.apply(alias, strings.ToLower(action), param)
	}
}

func (s *State) apply(alias, action, param string) {
	alias = strings.TrimSpace(alias)
	if alias == "" || action == "" {
		return
	}
	switch action {
	case "on", "off":
		s.RecordAction(alias, action)
		return
	case "color":
		color := strings.ToLower(strings.TrimSpace(param))
		if color == "" {
			color = action
		}
		s.setColor(alias, color)
		return
	case "brightness":
		if val, ok := parseInt(param); ok {
			s.setBrightness(alias, val)
		}
		return
	case "t", "temp", "temperature":
		if val, ok := parseInt(param); ok {
			s.setTemperature(alias, val)
		}
		return
	}

	// Fallbacks: handle numeric shorthand and color names.
	if val, ok := parseInt(action); ok {
		if s.setNumber(alias, val) {
			return
		}
	}
	if param != "" {
		if val, ok := parseInt(param); ok {
			if s.setNumber(alias, val) {
				return
			}
		}
	}
	s.setColor(alias, strings.ToLower(action))
}

func (s *State) setColor(alias, color string) {
	if color == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(alias)
	state.Color = color
	state.Power = ensureOn(state.Power)
	s.lights[alias] = state
}

func (s *State) setBrightness(alias string, value int) {
	if value < s.minBrightness || value > s.maxBrightness {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(alias)
	state.Brightness = value
	state.Power = ensureOn(state.Power)
	s.lights[alias] = state
}

func (s *State) setTemperature(alias string, value int) {
	if value < s.minTemperature || value > s.maxTemperature {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(alias)
	state.Temperature = value
	state.Power = ensureOn(state.Power)
	s.lights[alias] = state
}

func (s *State) setNumber(alias string, value int) bool {
	if value >= s.minBrightness && value <= s.maxBrightness {
		s.setBrightness(alias, value)
		return true
	}
	if value >= s.minTemperature && value <= s.maxTemperature {
		s.setTemperature(alias, value)
		return true
	}
	return false
}

func (s *State) ensure(alias string) LightState {
	if state, ok := s.lights[alias]; ok {
		return state
	}
	return LightState{}
}

func splitSceneCommand(raw string) (alias, action, param string) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return "", "", ""
	}
	alias = parts[0]
	if len(parts) > 1 {
		action = parts[1]
	}
	if len(parts) > 2 {
		param = parts[2]
	}
	return alias, action, param
}

func parseInt(val string) (int, bool) {
	if val == "" {
		return 0, false
	}
	num, err := strconv.Atoi(val)
	return num, err == nil
}

func ensureOn(current string) string {
	return "on"
}
