package tui

// Deps collects data and callbacks required by the TUI.
type Deps struct {
	Colors         []string
	Aliases        []string
	Scenes         []string
	ColorHex       func(string) (int64, bool)
	HexToRGB       func(int64) string
	ColorReset     string
	ValueSteps     int
	MinBrightness  int
	MaxBrightness  int
	MinTemperature int
	MaxTemperature int
	ResolveTargets func([]string) ([]string, error)
	Execute        func(ip, command, param string)
	RunScene       func(string) (string, error)
}

// defaults fills unset numeric fields with sensible defaults.
func (d *Deps) defaults() {
	if d.ValueSteps == 0 {
		d.ValueSteps = 10
	}
	if d.MinBrightness == 0 {
		d.MinBrightness = 5
	}
	if d.MaxBrightness == 0 {
		d.MaxBrightness = 100
	}
	if d.MinTemperature == 0 {
		d.MinTemperature = 1700
	}
	if d.MaxTemperature == 0 {
		d.MaxTemperature = 6500
	}
}
