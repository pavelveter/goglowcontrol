package main

import "time"

// Network parameters constants
const (
	// yeelightPort port for connecting to Yeelight devices
	yeelightPort = "55443"
	// connectionTimeout timeout for connecting to device
	connectionTimeout = 1200 * time.Millisecond
	// readTimeout timeout for reading response from device
	readTimeout = 1500 * time.Millisecond
	// sendTimeout timeout for sending command to device
	sendTimeout = 1 * time.Second
)

// Validation constants
const (
	// minBrightness minimum brightness value
	minBrightness = 1
	// maxBrightness maximum brightness value
	maxBrightness = 100
	// minTemperature minimum temperature value
	minTemperature = 1700
	// maxTemperature maximum temperature value
	maxTemperature = 6500
	// defaultDimValue brightness value when dimming
	defaultDimValue = 5
	// defaultBrightValue default brightness value
	defaultBrightValue = 100
	// defaultSmoothDuration default smooth transition duration
	defaultSmoothDuration = 500
)

// ANSI color codes
const (
	// colorReset reset color
	colorReset = "\033[0m"
	// colorBold bold font
	colorBold = "\033[1m"
	// colorRed red color
	colorRed = "\033[31m"
	// colorGreen green color
	colorGreen = "\033[32m"
	// colorYellow yellow color
	colorYellow = "\033[33m"
	// colorBlue blue color
	colorBlue = "\033[34m"
	// colorPurple purple color
	colorPurple = "\033[35m"
	// colorCyan cyan color
	colorCyan = "\033[36m"
)
