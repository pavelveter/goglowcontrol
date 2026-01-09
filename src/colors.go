package main

import (
	"strconv"
	"strings"
	"sync"
)

// colorCache cache for color parsing
type colorCache struct {
	// mu mutex for protecting cache access
	mu sync.RWMutex
	// colors color mapping
	colors map[string]int64
}

// colorMap global color cache instance
var colorMap colorCache

// get returns color from cache
func (c *colorCache) get(name string) (int64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.colors[strings.ToLower(name)]
	return val, ok
}

// load loads colors from string to cache
func (c *colorCache) load(colorsContent string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.colors = make(map[string]int64)

	for _, line := range strings.Split(strings.TrimSpace(colorsContent), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			continue
		}
		colorName := strings.TrimSpace(parts[0])
		hexValue := strings.TrimPrefix(strings.TrimSpace(parts[1]), "#")
		value, err := strconv.ParseInt(hexValue, 16, 32)
		if err != nil {
			continue // Skip invalid entries
		}
		c.colors[strings.ToLower(colorName)] = value
	}
	return nil
}

// colorToInt converts color name to its numeric value
func colorToInt(colorName string) (int64, bool) {
	return colorMap.get(colorName)
}
