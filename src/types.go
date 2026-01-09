package main

// Scene represents a scene with a set of commands
type Scene struct {
	// Name name of the scene
	Name string
	// Commands list of scene commands
	Commands []string
}

// Target defines the target channel for a command
type Target int

const (
	// Main main light
	Main Target = iota
	// BG ambient light
	BG
	// Both both channels
	Both
)

// DeviceCaps contains information about device capabilities
type DeviceCaps struct {
	// HasBG flag indicating presence of ambient channel
	HasBG bool
}
